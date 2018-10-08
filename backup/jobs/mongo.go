package jobs

import (
	"github.com/harishanchu/kube-backup/config"
	"time"
	"fmt"
	"github.com/codeskyblue/go-sh"
	"strings"
	"github.com/pkg/errors"
)

func RunMongoBackup(plan config.Plan, tmpPath string, filePostFix string) (string, string, error) {
	archive := fmt.Sprintf("%v/%v-%v.gz", tmpPath, plan.Name, filePostFix)
	log := fmt.Sprintf("%v/%v-%v.log", tmpPath, plan.Name, filePostFix)

	dump := fmt.Sprintf("mongodump --archive=%v --gzip --host %v --port %v ",
		archive, plan.Target["host"].(string), plan.Target["port"].(string))
	if plan.Target["database"] != "" {
		dump += fmt.Sprintf("--db %v ", plan.Target["database"])
	}
	if plan.Target["username"].(string) != "" && plan.Target["password"].(string) != "" {
		dump += fmt.Sprintf("-u %v -p %v ", plan.Target["username"].(string), plan.Target["password"].(string))
	}
	if plan.Target["params"].(string) != "" {
		dump += fmt.Sprintf("%v", plan.Target["params"].(string))
	}

	output, err := sh.Command("/bin/sh", "-c", dump).SetTimeout(time.Duration(plan.Scheduler.Timeout) * time.Minute).CombinedOutput()
	if err != nil {
		ex := ""
		if len(output) > 0 {
			ex = strings.Replace(string(output), "\n", " ", -1)
		}
		return "", "", errors.Wrapf(err, "mongodump log %v", ex)
	}
	logToFile(log, output)

	return archive, log, nil
}