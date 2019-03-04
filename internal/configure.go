package internal

import (
	"github.com/pkg/errors"
	"github.com/wal-g/wal-g/internal/storages/storage"
	"github.com/wal-g/wal-g/internal/tracelog"
	"golang.org/x/time/rate"
	"os"
	"path/filepath"
	"strconv"
)

const (
	DefaultDataBurstRateLimit = 8 * int64(DatabasePageSize)
	DefaultDataFolderPath     = "/tmp"
	WaleFileHost              = "file://localhost"
)

// TODO : unit tests
func configureLimiters() error {
	if diskLimitStr := getSettingValue("WALG_DISK_RATE_LIMIT"); diskLimitStr != "" {
		diskLimit, err := strconv.ParseInt(diskLimitStr, 10, 64)
		if err != nil {
			return errors.Wrap(err, "failed to parse WALG_DISK_RATE_LIMIT")
		}
		DiskLimiter = rate.NewLimiter(rate.Limit(diskLimit), int(diskLimit+DefaultDataBurstRateLimit)) // Add 8 pages to possible bursts
	}

	if netLimitStr := getSettingValue("WALG_NETWORK_RATE_LIMIT"); netLimitStr != "" {
		netLimit, err := strconv.ParseInt(netLimitStr, 10, 64)
		if err != nil {
			return errors.Wrap(err, "failed to parse WALG_NETWORK_RATE_LIMIT")
		}
		NetworkLimiter = rate.NewLimiter(rate.Limit(netLimit), int(netLimit+DefaultDataBurstRateLimit)) // Add 8 pages to possible bursts
	}
	return nil
}

// TODO : unit tests
func configureFolder() (storage.Folder, error) {
	skippedPrefixes := make([]string, 0)
	for _, adapter := range StorageAdapters {
		prefix := getSettingValue(adapter.prefixName)
		if prefix == "" {
			skippedPrefixes = append(skippedPrefixes, adapter.prefixName)
			continue
		}
		if adapter.prefixPreprocessor != nil {
			prefix = adapter.prefixPreprocessor(prefix)
		}
		return adapter.configureFolder(prefix, adapter.loadSettings())
	}
	return nil, NewUnsetEnvVarError(skippedPrefixes)
}

// TODO : unit tests
func getDataFolderPath() string {
	pgdata, ok := LookupConfigValue("PGDATA")
	var dataFolderPath string
	if !ok {
		dataFolderPath = DefaultDataFolderPath
	} else {
		dataFolderPath = filepath.Join(pgdata, "pg_wal")
		if _, err := os.Stat(dataFolderPath); err != nil {
			dataFolderPath = filepath.Join(pgdata, "pg_xlog")
			if _, err := os.Stat(dataFolderPath); err != nil {
				dataFolderPath = DefaultDataFolderPath
			}
		}
	}
	dataFolderPath = filepath.Join(dataFolderPath, "walg_data")
	return dataFolderPath
}

// TODO : unit tests
func configureWalDeltaUsage() (useWalDelta bool, deltaDataFolder DataFolder, err error) {
	if useWalDeltaStr, ok := LookupConfigValue("WALG_USE_WAL_DELTA"); ok {
		useWalDelta, err = strconv.ParseBool(useWalDeltaStr)
		if err != nil {
			return false, nil, errors.Wrapf(err, "failed to parse WALG_USE_WAL_DELTA")
		}
	}
	if !useWalDelta {
		return
	}
	dataFolderPath := getDataFolderPath()
	deltaDataFolder, err = NewDiskDataFolder(dataFolderPath)
	if err != nil {
		useWalDelta = false
		tracelog.WarningLogger.Printf("can't use wal delta feature because can't open delta data folder '%s'"+
			" due to error: '%v'\n", dataFolderPath, err)
		err = nil
	}
	return
}

// TODO : unit tests
func configureCompressor() (Compressor, error) {
	compressionMethod := getSettingValue("WALG_COMPRESSION_METHOD")
	if compressionMethod == "" {
		compressionMethod = Lz4AlgorithmName
	}
	if _, ok := Compressors[compressionMethod]; !ok {
		return nil, NewUnknownCompressionMethodError()
	}
	return Compressors[compressionMethod], nil
}

// TODO : unit tests
func configureLogging() error {
	logLevel, ok := LookupConfigValue("WALG_LOG_LEVEL")
	if ok {
		return tracelog.UpdateLogLevel(logLevel)
	}
	return nil
}

// Configure connects to storage and creates an uploader. It makes sure
// that a valid session has started; if invalid, returns AWS error
// and `<nil>` values.
func Configure() (uploader *Uploader, destinationFolder storage.Folder, err error) {
	err = configureLogging()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to configure logging")
	}

	err = configureLimiters()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to configure limiters")
	}

	folder, err := configureFolder()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to configure folder")
	}

	compressor, err := configureCompressor()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to configure compression")
	}

	useWalDelta, deltaDataFolder, err := configureWalDeltaUsage()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to configure WAL Delta usage")
	}

	preventWalOverwrite := false
	if preventWalOverwriteStr := getSettingValue("WALG_PREVENT_WAL_OVERWRITE"); preventWalOverwriteStr != "" {
		preventWalOverwrite, err = strconv.ParseBool(preventWalOverwriteStr)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to parse WALG_PREVENT_WAL_OVERWRITE")
		}
	}

	uploader = NewUploader(compressor, folder, deltaDataFolder, useWalDelta, preventWalOverwrite)

	return uploader, folder, err
}
