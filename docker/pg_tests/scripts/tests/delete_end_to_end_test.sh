#!/bin/sh
set -e -x
CONFIG_FILE="/tmp/configs/delete_end_to_end_test_config.json"
tmp/scripts/wrap_config_file.sh ${CONFIG_FILE}

/usr/lib/postgresql/10/bin/initdb ${PGDATA}

echo "archive_mode = on" >> /var/lib/postgresql/10/main/postgresql.conf
echo "archive_command = '/usr/bin/timeout 600 /usr/bin/wal-g --config=${CONFIG_FILE} wal-push %p'" >> /var/lib/postgresql/10/main/postgresql.conf
echo "archive_timeout = 600" >> /var/lib/postgresql/10/main/postgresql.conf

/usr/lib/postgresql/10/bin/pg_ctl -D ${PGDATA} -w start

pgbench -c 2 -T 100000000 -S || true &

for i in $(seq 1 9);
do
    pgbench -i -s 2 postgres
    if [ $i -eq 4 -o $i -eq 9 ];
    then
        pg_dumpall -f /tmp/dump$i
    fi
    sleep 1
    wal-g --config=${CONFIG_FILE} backup-push ${PGDATA}
done

wal-g --config=${CONFIG_FILE} backup-list

target_backup_name=`wal-g --config=${CONFIG_FILE} backup-list | tail -n 6 | head -n 1 | cut -f 1 -d " "`

wal-g --config=${CONFIG_FILE} delete before FIND_FULL $target_backup_name --confirm

wal-g --config=${CONFIG_FILE} backup-list

FIRST=`wal-g --config=${CONFIG_FILE} backup-list | head -n 2 | tail -n 1 | cut -f 1 -d " "`

for i in ${FIRST} LATEST
do
    tmp/scripts/drop_pg.sh
    wal-g --config=${CONFIG_FILE} backup-fetch ${PGDATA} ${i}
    echo "restore_command = 'echo \"WAL file restoration: %f, %p\"&& /usr/bin/wal-g --config=${CONFIG_FILE} wal-fetch \"%f\" \"%p\"'" > ${PGDATA}/recovery.conf
    /usr/lib/postgresql/10/bin/pg_ctl -D ${PGDATA} -w start
    wal-g --config=${CONFIG_FILE} backup-list
    sleep 10
    pg_dumpall -f /tmp/dump${i}
done

diff /tmp/dump4 /tmp/dump${FIRST}
diff /tmp/dump9 /tmp/dumpLATEST

tmp/scripts/drop_pg.sh

echo $target_backup_name
echo $FIRST
echo "End to end delete test success!!!!!!"
