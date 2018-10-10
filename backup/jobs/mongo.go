package jobs

import (
	"github.com/harishanchu/kube-backup/config"
	"time"
	"fmt"
	"github.com/codeskyblue/go-sh"
	"strings"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	k8corev1 "k8s.io/api/core/v1"
	k8clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func RunMongoBackup(plan config.Plan, k8Client *kubernetes.Clientset, tmpPath string, filePostFix string) (string, string, error) {
	archive := fmt.Sprintf("%v/%v-%v.gz", tmpPath, plan.Name, filePostFix)
	log := fmt.Sprintf("%v/%v-%v.log", tmpPath, plan.Name, filePostFix)
	k8coreV1Client := k8Client.CoreV1();

	username, password, _ := retrieveCredentials(k8coreV1Client, plan.Target["secret"])

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

func retrieveCredentials(k8Client k8clientcorev1.CoreV1Interface, secret interface{}) (string, string, error){
	username := ""
	password := ""
	secretMap := secret.(map[interface{}]interface{})

	secretName := secretMap["name"].(string)
	secretNameSpace := secretMap["namespace"].(string)
	usernmeItem := secretMap["usernameItem"].(string)
	passwordItem := secretMap["passwordItem"].(string)

	secret, err := k8Client.Secrets(secretNameSpace).Get(secretName, v1.GetOptions{})
	secretmap :=secret.(*k8corev1.Secret)

	if err != nil {
		return username, password, err
	}

	return string(secretmap.Data[usernmeItem]), string(secretmap.Data[passwordItem]), err
}