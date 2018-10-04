package jobs

import (
	"github.com/harishanchu/kube-db-backup/config"
	"time"
	"fmt"
	"github.com/codeskyblue/go-sh"
	"strings"
	"github.com/pkg/errors"
)

func RunMongoBackup(plan config.Plan, tmpPath string, ts time.Time) (string, string, error) {
	archive := fmt.Sprintf("%v/%v-%v.gz", tmpPath, plan.Name, formatTimeForFilePostFix(ts))
	log := fmt.Sprintf("%v/%v-%v.log", tmpPath, plan.Name, formatTimeForFilePostFix(ts))

	dump := fmt.Sprintf("mongodump --archive=%v --gzip --host %v --port %v ",
		archive, plan.Target["host"], plan.Target["port"])
	if plan.Target["database"] != "" {
		dump += fmt.Sprintf("--db %v ", plan.Target["database"])
	}
	if plan.Target["username"] != "" && plan.Target["password"] != "" {
		dump += fmt.Sprintf("-u %v -p %v ", plan.Target["username"], plan.Target["password"])
	}
	if plan.Target["params"] != "" {
		dump += fmt.Sprintf("%v", plan.Target["params"])
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

func formatTimeForFilePostFix(t time.Time) string {
	return t.Format("20060102030405000")
}