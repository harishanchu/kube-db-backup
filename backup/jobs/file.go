package jobs

import (
	"github.com/harishanchu/kube-backup/config"
	"fmt"
	"github.com/codeskyblue/go-sh"
	"strings"
)

func RunFileBackup(plan config.Plan, tmpPath string, filePostFix string) (string, string, error) {
	backupLocation := fmt.Sprintf("%v/%v-%v", tmpPath, plan.Name, filePostFix)
	archive := backupLocation + ".gz"
	log := fmt.Sprintf("%v/%v-%v.log", tmpPath, plan.Name, filePostFix)

	err := RetrieveFiles(plan.Target["podLabels"].(string), plan.Target["namespace"].(string),
		plan.Target["paths"].([]interface{}), backupLocation, log)

	if (err != nil) {
		return archive, log, err
	}

	return archive, log, nil
}

func RetrieveFiles(podLabels, namespace string, filePaths []interface{}, backupLocation, logFile string) error {
	pods, _ := GetPods(podLabels, namespace, logFile)

	for _, pod := range pods {
		retrieveFileCommand := fmt.Sprintf("kubectl -n %v cp %v:%v %v", namespace, pod)

		for _, v := range filePaths {
			retrieveFileCommand = fmt.Sprintf(retrieveFileCommand, v.(string), backupLocation)

			output, err := sh.Command("sh", "-c", retrieveFileCommand).CombinedOutput()

			if err != nil {
				return err;
			}

			logToFile(logFile, output)
		}
	}

	return nil;
}

func GetPods(podLabels, namespace, logFile string) ([]string, error) {
	labelsArray := strings.Split(podLabels, " ")
	listPodCommands := fmt.Sprintf("kubectl -n %v get pods -o go-template --template '{{range .items}}"+
		"{{.metadata.name}}{{\" \"}}{{end}}'", namespace)

	for _, label := range labelsArray {
		listPodCommands = fmt.Sprintf(listPodCommands+" -l %v", label)
	}

	output, err := sh.Command("sh", "-c", listPodCommands).CombinedOutput()

	if err != nil {
		return nil, err;
	}

	outputString := string(output)

	if(len(outputString) == 0) {
		return make([]string, 0), nil
	}

	logToFile(logFile, output)

	return strings.Split(outputString, " "), nil
}
