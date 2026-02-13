package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/spf13/cobra"
)

const (
	s3PartSize      = 64 * 1024 * 1024 // 64MB per part
	s3MaxConcurrent = 5                 // concurrent upload parts
)

// pipelineConfig holds all env-var configuration for the pipeline command.
type pipelineConfig struct {
	S3Bucket           string
	S3Prefix           string
	StorageClass       string
	CreateLatestLink   bool
	CleanupOldVersions bool
	KeepVersions       int
	Workdir            string
	Mode               string
	SkipVacuum         bool
}

var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Executa o pipeline completo: download → extract → load → upload S3",
	Long: `Pipeline completo automatizado para ECS Fargate Scheduled Task.
Configuração via variáveis de ambiente (zero parâmetros):

  S3_BUCKET            (obrigatório) Bucket S3 de destino
  S3_PREFIX            (default: cnpj) Prefixo no S3
  STORAGE_CLASS        (default: INTELLIGENT_TIERING) Classe de armazenamento
  CREATE_LATEST_LINK   (default: true) Cria link "latest" via CopyObject
  CLEANUP_OLD_VERSIONS (default: false) Remove versões antigas do S3
  KEEP_VERSIONS        (default: 3) Número de versões a manter
  BCD_WORKDIR          (default: /tmp/cnpj_rf) Diretório de trabalho
  BCD_MODE             (default: zip) Modo de download: zip ou tar
  BCD_SKIP_VACUUM      (default: false) Pular VACUUM do SQLite`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPipeline()
	},
}

func init() { RootCmd.AddCommand(pipelineCmd) }

func runPipeline() error {
	start := time.Now()

	// 1. Load config from env vars
	cfg, err := loadPipelineConfig()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// Apply config to global flags so existing commands work
	flagWorkdir = cfg.Workdir
	flagMode = cfg.Mode
	flagSkipVacuum = cfg.SkipVacuum
	flagOutDB = filepath.Join(cfg.Workdir, "cnpj.sqlite")

	logger.Info("pipeline starting",
		slog.String("workdir", cfg.Workdir),
		slog.String("mode", cfg.Mode),
		slog.String("s3_bucket", cfg.S3Bucket),
		slog.String("s3_prefix", cfg.S3Prefix),
	)

	if err := os.MkdirAll(cfg.Workdir, 0o755); err != nil {
		return fmt.Errorf("mkdir workdir: %w", err)
	}

	// 2. Download
	logger.Info("step 1/5: downloading data from Receita Federal")
	if cfg.Mode == "tar" {
		if err := downloadTar(); err != nil {
			return fmt.Errorf("download: %w", err)
		}
	} else {
		if err := downloadZips(); err != nil {
			return fmt.Errorf("download: %w", err)
		}
	}
	logger.Info("download completed", slog.String("duration", time.Since(start).String()))

	// 3. Extract
	extractStart := time.Now()
	logger.Info("step 2/5: extracting archives")
	if cfg.Mode == "tar" {
		if err := extractTar(); err != nil {
			return fmt.Errorf("extract: %w", err)
		}
	} else {
		if err := extractZips(); err != nil {
			return fmt.Errorf("extract: %w", err)
		}
	}
	logger.Info("extract completed", slog.String("duration", time.Since(extractStart).String()))

	// 4. Load into SQLite
	loadStart := time.Now()
	logger.Info("step 3/5: loading data into SQLite")
	ctx := context.Background()
	db, err := openSQLite(flagOutDB)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	if err := createSchema(ctx, db); err != nil {
		db.Close()
		return fmt.Errorf("schema: %w", err)
	}
	extracted := filepath.Join(cfg.Workdir, "extracted")
	if err := loadAll(ctx, db, extracted); err != nil {
		db.Close()
		return fmt.Errorf("load: %w", err)
	}
	if err := createIndexes(ctx, db); err != nil {
		db.Close()
		return fmt.Errorf("indexes: %w", err)
	}
	if err := showStats(ctx, db); err != nil {
		db.Close()
		return fmt.Errorf("stats: %w", err)
	}
	db.Close()
	logger.Info("load completed", slog.String("duration", time.Since(loadStart).String()))

	// 5. Upload to S3
	uploadStart := time.Now()
	logger.Info("step 4/5: uploading to S3")
	ym := flagYearMonth
	if ym == "" {
		ym = time.Now().Format("2006-01")
	}
	s3Key := fmt.Sprintf("%s/%s/cnpj.sqlite", cfg.S3Prefix, ym)
	if err := uploadToS3(ctx, cfg, flagOutDB, s3Key); err != nil {
		return fmt.Errorf("s3 upload: %w", err)
	}
	logger.Info("upload completed",
		slog.String("key", s3Key),
		slog.String("duration", time.Since(uploadStart).String()),
	)

	// Create "latest" link via server-side copy
	if cfg.CreateLatestLink {
		latestKey := fmt.Sprintf("%s/latest/cnpj.sqlite", cfg.S3Prefix)
		logger.Info("creating latest link", slog.String("key", latestKey))
		if err := s3CopyObject(ctx, cfg, s3Key, latestKey); err != nil {
			logger.Warn("failed to create latest link (non-fatal)", slog.String("error", err.Error()))
		}
	}

	// Cleanup old S3 versions
	if cfg.CleanupOldVersions {
		logger.Info("cleaning up old S3 versions", slog.Int("keep", cfg.KeepVersions))
		if err := cleanupOldS3Versions(ctx, cfg); err != nil {
			logger.Warn("cleanup old versions failed (non-fatal)", slog.String("error", err.Error()))
		}
	}

	// 6. Cleanup local workdir
	logger.Info("step 5/5: cleaning up local files")
	cleanupWorkdir(cfg.Workdir, flagOutDB)

	logger.Info("pipeline completed successfully",
		slog.String("total_duration", time.Since(start).String()),
		slog.String("version", ym),
		slog.String("s3_key", s3Key),
	)
	return nil
}

func loadPipelineConfig() (*pipelineConfig, error) {
	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		return nil, fmt.Errorf("S3_BUCKET environment variable is required")
	}

	keepVersions, _ := strconv.Atoi(getEnvOr("KEEP_VERSIONS", "3"))
	if keepVersions < 1 {
		keepVersions = 3
	}

	return &pipelineConfig{
		S3Bucket:           bucket,
		S3Prefix:           getEnvOr("S3_PREFIX", "cnpj"),
		StorageClass:       getEnvOr("STORAGE_CLASS", "INTELLIGENT_TIERING"),
		CreateLatestLink:   getEnvBool("CREATE_LATEST_LINK", true),
		CleanupOldVersions: getEnvBool("CLEANUP_OLD_VERSIONS", false),
		KeepVersions:       keepVersions,
		Workdir:            getEnvOr("BCD_WORKDIR", "/tmp/cnpj_rf"),
		Mode:               getEnvOr("BCD_MODE", "zip"),
		SkipVacuum:         getEnvBool("BCD_SKIP_VACUUM", false),
	}, nil
}

func uploadToS3(ctx context.Context, cfg *pipelineConfig, localPath, key string) error {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("aws config: %w", err)
	}

	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", localPath, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}
	logger.Info("uploading file",
		slog.String("file", localPath),
		slog.Float64("size_gb", float64(info.Size())/(1024*1024*1024)),
		slog.String("bucket", cfg.S3Bucket),
		slog.String("key", key),
	)

	client := s3.NewFromConfig(awsCfg)
	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = s3PartSize
		u.Concurrency = s3MaxConcurrent
	})

	storageClass := types.StorageClass(cfg.StorageClass)

	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(cfg.S3Bucket),
		Key:          aws.String(key),
		Body:         f,
		StorageClass: storageClass,
	})
	if err != nil {
		return fmt.Errorf("s3 upload: %w", err)
	}

	logger.Info("upload successful", slog.String("key", key))
	return nil
}

func s3CopyObject(ctx context.Context, cfg *pipelineConfig, srcKey, dstKey string) error {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)
	copySource := fmt.Sprintf("%s/%s", cfg.S3Bucket, srcKey)
	storageClass := types.StorageClass(cfg.StorageClass)

	_, err = client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:            aws.String(cfg.S3Bucket),
		CopySource:        aws.String(copySource),
		Key:               aws.String(dstKey),
		StorageClass:      storageClass,
		MetadataDirective: types.MetadataDirectiveReplace,
	})
	if err != nil {
		return fmt.Errorf("s3 copy: %w", err)
	}

	logger.Info("latest link created", slog.String("key", dstKey))
	return nil
}

func cleanupOldS3Versions(ctx context.Context, cfg *pipelineConfig) error {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)
	prefix := cfg.S3Prefix + "/"

	resp, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(cfg.S3Bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return fmt.Errorf("list objects: %w", err)
	}

	// Collect YYYY-MM version prefixes (skip "latest")
	var versions []string
	for _, cp := range resp.CommonPrefixes {
		p := aws.ToString(cp.Prefix)
		name := strings.TrimPrefix(p, prefix)
		name = strings.TrimSuffix(name, "/")
		if ymRe.MatchString(name) {
			versions = append(versions, name)
		}
	}

	// Sort descending (newest first) and remove old ones
	sortDescending(versions)

	if len(versions) <= cfg.KeepVersions {
		logger.Info("no old versions to clean up", slog.Int("total", len(versions)))
		return nil
	}

	toDelete := versions[cfg.KeepVersions:]
	for _, ver := range toDelete {
		verPrefix := fmt.Sprintf("%s/%s/", cfg.S3Prefix, ver)
		logger.Info("deleting old version", slog.String("prefix", verPrefix))

		listResp, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String(cfg.S3Bucket),
			Prefix: aws.String(verPrefix),
		})
		if err != nil {
			logger.Warn("failed to list objects for deletion", slog.String("prefix", verPrefix), slog.String("error", err.Error()))
			continue
		}

		for _, obj := range listResp.Contents {
			_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
				Bucket: aws.String(cfg.S3Bucket),
				Key:    obj.Key,
			})
			if err != nil {
				logger.Warn("failed to delete object", slog.String("key", aws.ToString(obj.Key)), slog.String("error", err.Error()))
			}
		}
		logger.Info("deleted old version", slog.String("version", ver))
	}

	return nil
}

func sortDescending(ss []string) {
	// Simple insertion sort for small slices (typically < 20 versions)
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j] > ss[j-1]; j-- {
			ss[j], ss[j-1] = ss[j-1], ss[j]
		}
	}
}

func cleanupWorkdir(workdir, dbPath string) {
	if err := os.RemoveAll(workdir); err != nil {
		logger.Warn("failed to cleanup workdir (non-fatal)", slog.String("error", err.Error()))
	} else {
		logger.Info("workdir cleaned", slog.String("path", workdir))
	}

	// Remove SQLite if it's outside workdir
	if !strings.HasPrefix(dbPath, workdir) {
		if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
			logger.Warn("failed to remove sqlite (non-fatal)", slog.String("error", err.Error()))
		}
	}
}

func getEnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	v = strings.ToLower(v)
	return v == "true" || v == "1" || v == "yes"
}
