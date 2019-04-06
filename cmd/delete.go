package cmd

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/wal-g/wal-g/internal"
	"github.com/wal-g/wal-g/internal/storages/storage"
	"github.com/wal-g/wal-g/internal/tracelog"
	"regexp"
	"strconv"
	"time"
)

const (
	ConfirmFlag            = "confirm"
	DeleteShortDescription = "Clears old backups and WALs"

	DeleteRetainExamples = `  retain 5                      keep 5 backups
  retain FULL 5                 keep 5 full backups and all deltas of them
  retain FIND_FULL 5            find necessary full for 5th and keep everything after it`

	DeleteBeforeExamples = `  before base_0123              keep everything after base_0123 including itself
  before FIND_FULL base_0123    keep everything after the base of base_0123`
)

var StringModifiers = []string{"FULL", "FIND_FULL"}

var confirmed = false
var patternLSN = "[0-9A-F]{24}"
var patternBackupName = fmt.Sprintf("base_%[1]s(_D_%[1]s)?", patternLSN)
var regexpLSN = regexp.MustCompile(patternLSN)
var regexpBackupName = regexp.MustCompile(patternBackupName)
var maxCountOfLSN = 2

func extractDeleteModifierFromArgs(args []string) (int, string) {
	if len(args) == 1 {
		return internal.NoDeleteModifier, args[0]
	} else if args[0] == StringModifiers[internal.FullDeleteModifier-1] {
		return internal.FullDeleteModifier, args[1]
	} else {
		return internal.FindFullDeleteModifier, args[1]
	}
}

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: DeleteShortDescription, // TODO : improve description
}

func deleteArgsValidator(cmd *cobra.Command, args []string) error {
	if len(args) != 1 && len(args) != 2 {
		return fmt.Errorf("accepts between 1 and 2 arg(s), received %d", len(args))
	}
	if len(args) == 2 {
		expectedModifier := args[0]
		if expectedModifier != StringModifiers[0] && expectedModifier != StringModifiers[1] {
			return fmt.Errorf("expected to get one of modifiers: %v as first argument", StringModifiers)
		}
	}
	return nil
}

func deleteRetainArgsValidator(cmd *cobra.Command, args []string) error {
	err := deleteArgsValidator(cmd, args)
	if err != nil {
		return err
	}
	_, retantionStr := extractDeleteModifierFromArgs(args)
	retantionNumber, err := strconv.Atoi(retantionStr)
	if err != nil {
		return errors.Wrapf(err, "expected to get a number as retantion count, but got: '%s'", retantionStr)
	}
	if retantionNumber <= 0 {
		return fmt.Errorf("cannot retain less than one backup") // TODO : Consider allowing to delete everything
	}
	return nil
}

func runDeleteRetain(cmd *cobra.Command, args []string) {
	modifier, retantionStr := extractDeleteModifierFromArgs(args)
	folder, err := internal.ConfigureFolder()
	if err != nil {
		tracelog.ErrorLogger.FatalError(err)
	}
	retantionCount, _ := strconv.Atoi(retantionStr)
	internal.HandleDeleteRetain(folder, retantionCount, modifier, !confirmed)
}

var deleteRetainCmd = &cobra.Command{
	Use:       "retain [FULL|FIND_FULL] backup_count", // TODO : improve description
	Example:   DeleteRetainExamples,
	ValidArgs: StringModifiers,
	Args:      deleteRetainArgsValidator,
	Run:       runDeleteRetain,
}

func deleteBeforeArgsValidator(cmd *cobra.Command, args []string) error {
	err := deleteArgsValidator(cmd, args)
	if err != nil {
		return err
	}
	modifier, beforeStr := extractDeleteModifierFromArgs(args)
	if modifier == internal.FullDeleteModifier {
		return fmt.Errorf("unsupported moodifier for delete before command")
	}
	if before, err := time.Parse(time.RFC3339, beforeStr); err == nil {
		if before.After(time.Now()) {
			return fmt.Errorf("cannot delete before future date")
		}
	}
	return nil
}

func runDeleteBefore(cmd *cobra.Command, args []string) {
	modifier, beforeStr := extractDeleteModifierFromArgs(args)
	folder, err := internal.ConfigureFolder()
	if err != nil {
		tracelog.ErrorLogger.FatalError(err)
	}
	before, err := time.Parse(time.RFC3339, beforeStr)
	if err == nil {
		potentialTarget, err := internal.FindFirstLaterOrEqualTime(folder, before, postgresLess)
		if err != nil {
			tracelog.ErrorLogger.FatalError(err)
		}
		beforeStr = fetchBackupName(potentialTarget)
	}
	target, err := internal.FindTargetBeforeName(folder, beforeStr, modifier)
	if err != nil {
		tracelog.ErrorLogger.FatalError(err)
	}
	if confirmed {
		err = storage.DeleteObjectsWhere(folder, func(object storage.Object) bool {
			return postgresLess(object, target)
		})
	}
}

var deleteBeforeCmd = &cobra.Command{
	Use:     "before [FIND_FULL] backup_name|timestamp", // TODO : improve description
	Example: DeleteBeforeExamples,
	Args:    deleteBeforeArgsValidator,
	Run:     runDeleteBefore,
}

func init() {
	RootCmd.AddCommand(deleteCmd)

	deleteCmd.AddCommand(deleteRetainCmd, deleteBeforeCmd)

	deleteCmd.PersistentFlags().BoolVar(&confirmed, ConfirmFlag, false, "Confirms backup deletion")
}

// it's here because it's part of postgres logic and in future it will be placed in postgres part of wal-g
func postgresLess(object1 storage.Object, object2 storage.Object) bool {
	return fetchLSN(object1) < fetchLSN(object2)
}

func fetchLSN(object storage.Object) string {
	return regexpLSN.FindAllString(object.GetName(), maxCountOfLSN)[0]
}

func fetchBackupName(object storage.Object) string {
	return regexpBackupName.FindString(object.GetName())
}
