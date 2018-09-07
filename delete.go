package walg

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"log"
	"strconv"
	"strings"
	"time"
)

const DeleteUsageText = "delete requires at least 2 parameters" + `
		retain 5                      keep 5 backups
		retain FULL 5                 keep 5 full backups and all deltas of them
		retail FIND_FULL 5            find necessary full for 5th and keep everything after it
		before base_0123              keep everything after base_0123 including itself
		before FIND_FULL base_0123    keep everything after the base of base_0123`

// DeleteCommandArguments incapsulates arguments for delete command
type DeleteCommandArguments struct {
	Full       bool
	FindFull   bool
	Retain     bool
	Before     bool
	Target     string
	BeforeTime *time.Time
	dryrun     bool
}

// ParseDeleteArguments interprets arguments for delete command. TODO: use flags or cobra
func ParseDeleteArguments(args []string, fallBackFunc func()) (result DeleteCommandArguments) {
	if len(args) < 3 {
		fallBackFunc()
		return
	}

	params := args[1:]
	if params[0] == "retain" {
		result.Retain = true
		params = params[1:]
	} else if params[0] == "before" {
		result.Before = true
		params = params[1:]
	} else {
		fallBackFunc()
		return
	}
	if params[0] == "FULL" {
		result.Full = true
		params = params[1:]
	} else if params[0] == "FIND_FULL" {
		result.FindFull = true
		params = params[1:]
	}
	if len(params) < 1 {
		log.Print("Backup name not specified")
		fallBackFunc()
		return
	}

	result.Target = params[0]
	if t, err := time.Parse(time.RFC3339, result.Target); err == nil {
		if t.After(time.Now()) {
			log.Println("Cannot delete before future date")
			fallBackFunc()
		}
		result.BeforeTime = &t
	}
	//if DeleteConfirmed && !DeleteDryrun  // TODO: use flag
	result.dryrun = true
	if len(params) > 1 && (params[1] == "--confirm" || params[1] == "-confirm") {
		result.dryrun = false
	}

	if result.Retain {
		number, err := strconv.Atoi(result.Target)
		if err != nil {
			log.Println("Cannot parse target number ", number)
			fallBackFunc()
			return
		}
		if number <= 0 {
			log.Println("Cannot retain 0") // Consider allowing to delete everything
			fallBackFunc()
			return
		}
	}
	return
}

// TODO : unit tests
func deleteBeforeTarget(target *Backup, findFull bool, backups []BackupTime, dryRun bool) {
	folder := target.Folder
	dto := target.fetchSentinel()
	if dto.isIncremental() {
		if findFull {
			target.Name = *dto.IncrementFullName
		} else {
			log.Fatalf("%v is incemental and it's predecessors cannot be deleted. Consider FIND_FULL option.", target.Name)
		}
	}
	var err error
	if backups == nil {
		backups, err = getBackups(folder)
		if err != nil {
			log.Fatal(err)
		}
	}

	skip := true
	skipLine := len(backups)
	for i, b := range backups {
		if skip {
			log.Printf("%v skipped\n", b.Name)
		} else {
			log.Printf("%v will be deleted\n", b.Name)
		}
		if b.Name == target.Name {
			skip = false
			skipLine = i
		}
	}

	if !dryRun {
		if skipLine < len(backups)-1 {
			deleteWALBefore(backups[skipLine], folder)
			deleteBackupsBefore(backups, skipLine, folder)
		}
	} else {
		log.Printf("Dry run finished.\n")
	}
}

// TODO : unit tests
func deleteBackupsBefore(backups []BackupTime, skipline int, folder *S3Folder) {
	for i, b := range backups {
		if i > skipline {
			dropBackup(folder, b)
		}
	}
}

// TODO : unit tests
func dropBackup(folder *S3Folder, backupTime BackupTime) {
	backup := NewBackup(folder, backupTime.Name)
	tarFiles, err := backup.GetKeys()
	if err != nil {
		log.Fatal("Unable to list backup for deletion ", backupTime.Name, err)
	}

	folderKey := strings.TrimPrefix(GetBackupPath(folder)+backupTime.Name, "/")
	sentinelKey := folderKey + SentinelSuffix

	keys := append(tarFiles, sentinelKey, folderKey)
	parts := partition(keys, 1000)
	for _, part := range parts {

		input := &s3.DeleteObjectsInput{Bucket: folder.Bucket, Delete: &s3.Delete{
			Objects: partitionToObjects(part),
		}}
		_, err = folder.S3API.DeleteObjects(input)
		if err != nil {
			log.Fatal("Unable to delete backup ", backupTime.Name, err)
		}

	}
}

// TODO : unit tests
func partitionToObjects(keys []string) []*s3.ObjectIdentifier {
	objs := make([]*s3.ObjectIdentifier, len(keys))
	for i, k := range keys {
		objs[i] = &s3.ObjectIdentifier{Key: aws.String(k)}
	}
	return objs
}

// TODO : unit tests
func deleteWALBefore(backupTime BackupTime, folder *S3Folder) {
	objects, err := getWals(backupTime.WalFileName, folder)
	if err != nil {
		log.Fatal("Unable to obtaind WALS for border ", backupTime.Name, err)
	}
	parts := partitionObjects(objects, 1000)
	for _, part := range parts {
		input := &s3.DeleteObjectsInput{Bucket: folder.Bucket, Delete: &s3.Delete{
			Objects: part,
		}}
		_, err = folder.S3API.DeleteObjects(input)
		if err != nil {
			log.Fatal("Unable to delete WALS before ", backupTime.Name, err)
		}
	}
}

func printDeleteUsageAndFail() {
	log.Fatal(DeleteUsageText)
}
