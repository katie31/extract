package main

import (
	mysql_cmd "github.com/wal-g/wal-g/cmd/mysql"
	"github.com/wal-g/wal-g/internal/databases/mysql"
	"github.com/wal-g/wal-g/internal"
	"github.com/wal-g/wal-g/main"
)

func main() {
	internal.UpdateAllowedConfig(mysql.AllowedConfigKeys)
	config.Configure()
	mysql_cmd.Execute()
}
