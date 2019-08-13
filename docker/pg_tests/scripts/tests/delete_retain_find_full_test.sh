#!/bin/sh
set -e -x
CONFIG_FILE="/tmp/configs/delete_retain_find_full_test_config.json"

COMMON_CONFIG="/tmp/configs/common_config.json"
TMP_CONFIG="/tmp/configs/tmp_config.json"
cat ${CONFIG_FILE} > ${TMP_CONFIG}
echo "," >> ${TMP_CONFIG}
cat ${COMMON_CONFIG} >> ${TMP_CONFIG}

tmp/scripts/wrap_config_file.sh ${TMP_CONFIG}
cat ${TMP_CONFIG}

WAL_PUSH_LOGS="/tmp/logs/wal_push_logs/pg_delete_retain_find_full_test_logs"
WAL_FETCH_LOGS="/tmp/logs/wal_fetch_logs/pg_delete_retain_find_full_test_logs"
BACKUP_PUSH_LOGS="/tmp/logs/backup_push_logs/pg_delete_retain_find_full_test_logs"
BACKUP_FETCH_LOGS="/tmp/logs/backup_fetch_logs/pg_delete_retain_find_full_test_logs"

/usr/lib/postgresql/10/bin/initdb ${PGDATA}

echo "archive_mode = on" >> /var/lib/postgresql/10/main/postgresql.conf
echo "archive_command = '/usr/bin/timeout 600 /usr/bin/time -v -a --output ${WAL_PUSH_LOGS} /usr/bin/wal-g --config=${TMP_CONFIG} wal-push %p'" >> /var/lib/postgresql/10/main/postgresql.conf
echo "archive_timeout = 600" >> /var/lib/postgresql/10/main/postgresql.conf

/usr/lib/postgresql/10/bin/pg_ctl -D ${PGDATA} -w start

for i in 1 2 3 4 5 6
do
    pgbench -i -s 1 postgres &
    sleep 1
    /usr/bin/time -v -a --output ${BACKUP_PUSH_LOGS} wal-g --config=${TMP_CONFIG} backup-push ${PGDATA}
done

wal-g --config=${TMP_CONFIG} backup-list
lines_before_delete=`wal-g --config=${TMP_CONFIG} backup-list | wc -l`
wal-g --config=${TMP_CONFIG} backup-list | tail -n 4 > /tmp/list_tail_before_delete

wal-g --config=${TMP_CONFIG} delete retain FIND_FULL 3 --confirm

wal-g --config=${TMP_CONFIG} backup-list
lines_after_delete=`wal-g --config=${TMP_CONFIG} backup-list | wc -l`
wal-g --config=${TMP_CONFIG} backup-list | tail -n 4 > /tmp/list_tail_after_delete

if [ $(($lines_before_delete-2)) -ne $lines_after_delete ];
then
    echo $(($lines_before_delete-2)) > /tmp/before_delete
    echo $lines_after_delete > /tmp/after_delete
    echo "Wrong number of deleted lines"
    diff /tmp/before_delete /tmp/after_delete
fi

diff /tmp/list_tail_before_delete /tmp/list_tail_after_delete

tmp/scripts/drop_pg.sh
rm ${TMP_CONFIG}
echo "Delete retain FIND_FULL success!!!!!!"
