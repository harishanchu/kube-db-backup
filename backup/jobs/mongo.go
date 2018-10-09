package jobs

import (
	"github.com/harishanchu/kube-backup/config"
	"time"
	"fmt"
	"github.com/codeskyblue/go-sh"
	"strings"
	"github.com/pkg/errors"
	"encoding/json"
)

func RunMongoBackup(plan config.Plan, tmpPath string, filePostFix string) (string, string, error) {
	archive := fmt.Sprintf("%v/%v-%v.gz", tmpPath, plan.Name, filePostFix)
	log := fmt.Sprintf("%v/%v-%v.log", tmpPath, plan.Name, filePostFix)

	username, password, _ := retrieveCredentials(plan.Target["secret"])

	dump := fmt.Sprintf("mongodump --archive=%v --gzip --host %v --port %v ",
		archive, plan.Target["host"].(string), plan.Target["port"].(string))
	if plan.Target["database"] != "" {
		dump += fmt.Sprintf("--db %v ", plan.Target["database"])
	}
	if username != "" && password != "" {
		dump += fmt.Sprintf("-u %v -p %v ", username, password)
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

func retrieveCredentials(secret interface{}) (string, string, error){
	username := ""
	password := ""
	secretMap := secret.(map[interface{}]interface{})

	secretName := secretMap["name"].(string)
	secretNameSpace := secretMap["namespace"].(string)
	//usernmeItem := secretMap["usernmeItem"].(string)
	//passwordItem := secretMap["passwordItem"].(string)

	retrieveSecretCommand := fmt.Sprintf("echo '{';"+
	"for row in $(kubectl get secret %v -o json -n %v | jq -c '.data | to_entries[]'); do "+
	"KEY=$(echo ${row} | jq -r '.key');"+
	"DECODED=$(echo ${row} | jq -r '.value' | base64 --decode);"+
	"echo \"\\\"$KEY\\\": \\\"$DECODED\\\",\";"+
	"done;"+
	"echo '}';", secretName, secretNameSpace)

	output, err := sh.Command("sh", "-c", retrieveSecretCommand).CombinedOutput()

	if err != nil {
		return username, password, err
	}

	parsedSecret := new(interface{})

	err = json.Unmarshal(output, &parsedSecret)

	if err != nil {
		return username, password, err
	}

	return username, password, err
}