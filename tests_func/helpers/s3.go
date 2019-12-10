package helpers

import (
	"fmt"
	"github.com/docker/docker/api/types"
	testUtils "github.com/wal-g/wal-g/tests_func/utils"
	"strings"
)

func ConfigureS3(testContext *TestContextType, containerName *types.Container) error {
	var response string
	for i := 0; i < 100; i++ {
		bucketName := testUtils.GetVarFromEnvList(testContext.Env, "S3_BUCKET")
		accessKeyId := testUtils.GetVarFromEnvList(testContext.Env, "S3_ACCESS_KEY_ID")
		accessSecretKey := testUtils.GetVarFromEnvList(testContext.Env, "S3_ACCESS_SECRET_KEY")
		command := []string{"mc", "--debug", "config", "host", "add", "local", "http://localhost:9000", accessKeyId, accessSecretKey}
		response, _ = RunCommandInContainer(testContext, containerName.Names[0], command)

		command = []string{"mc", "mb", fmt.Sprintf("local/%s", bucketName)}
		response, _ = RunCommandInContainer(testContext, containerName.Names[0], command)

		if strings.Contains(response, "created successfully") ||
			strings.Contains(response, "already own it") {
			return nil
		}
	}
	if !strings.Contains(response, "created successfully") {
		return fmt.Errorf("s3 is not available: %s", response)
	}
	return nil
}
