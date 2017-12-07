package walg

import (
	"log"
	"regexp"

	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

// Connect establishes a connection to postgres using
// a UNIX socket. Must export PGHOST and run with `sudo -E -u postgres`.
// If PGHOST is not set or if the connection fails, an error is returned
// and the connection is `<nil>`.
func Connect() (*pgx.Conn, error) {
	config, err := pgx.ParseEnvLibpq()
	if err != nil {
		return nil, errors.Wrap(err, "Connect: unable to read environment variables")
	}

	conn, err := pgx.Connect(config)
	if err != nil {
		return nil, errors.Wrap(err, "Connect: postgres connection failed")
	}

	return conn, nil
}

// StartBackup starts a non-exclusive base backup immediately. When finishing the backup,
// `backup_label` and `tablespace_map` contents are not immediately written to
// a file but returned instead. Returns empty string and an error if backup
// fails.
func (b *Bundle) StartBackup(conn *pgx.Conn, backup string) (string, uint64, error) {
	var name, lsnStr string
	var version int
	// We extract here version since it is not used elsewhere. If reused, this should be refactored.
	// TODO: implement offline backups, incapsulate PostgreSQL version logic and create test specs for this logic.
	// Currently all version-dependent logic is here
	err := conn.QueryRow("select (current_setting('server_version_num'))::int").Scan(&version)
	if err != nil {
		return "", 0, errors.Wrap(err, "QueryFile: getting Postgres version failed")
	}
	walname := "xlog"
	if version >= 100000 {
		walname = "wal"
	}

	query := "SELECT case when pg_is_in_recovery() then '' else (pg_" + walname + "file_name_offset(lsn)).file_name end, lsn::text, pg_is_in_recovery() FROM pg_start_backup($1, true, false) lsn"
	err = conn.QueryRow(query, backup).Scan(&name, &lsnStr, &b.Replica)

	lsn, err := ParseLsn(lsnStr)
	if err != nil {
		return "", 0, err
	}

	if b.Replica {
		name, b.Timeline, err = WALFileName(lsn, conn)
		if err != nil {
			return "", 0, err
		}
	}
	return backupNamePrefix + name, lsn, nil

}

const backupNamePrefix = "base_"

func (b *Bundle) CheckTimelineChanged(conn *pgx.Conn) bool {
	if b.Replica {
		timeline, err := readTimeline(conn)
		if err != nil {
			log.Printf("Unbale to check timeline change. Sentinel for the backup will not be uploaded.")
			return true
		}

		// Per discussion in
		// https://www.postgresql.org/message-id/flat/BF2AD4A8-E7F5-486F-92C8-A6959040DEB6%40yandex-team.ru#BF2AD4A8-E7F5-486F-92C8-A6959040DEB6@yandex-team.ru
		// Following check is the very pessimistic approach on replica backup invalidation
		if timeline != b.Timeline {
			log.Printf("Timeline has changed since backup start. Sentinel for the backup will not be uploaded.")
			return true
		}
	}
	return false
}

// FormatName grabs the name of the WAL file and returns it in the form of `base_...`.
// If no match is found, returns an empty string and a `NoMatchAvailableError`.
func FormatName(s string) (string, error) {
	re := regexp.MustCompile(`\(([^\)]+)\)`)
	f := re.FindString(s)
	if f == "" {
		return "", errors.Wrap(NoMatchAvailableError{s}, "FormatName:")
	}
	return backupNamePrefix + f[6:len(f)-1], nil
}
