package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/codeskyblue/go-sh"
	"github.com/harishanchu/kube-backup/config"
	"github.com/pkg/errors"
	"github.com/harishanchu/kube-backup/backup/jobs"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/kubernetes"
	"flag"
	"k8s.io/client-go/tools/clientcmd"
)

func Run(plan config.Plan, tmpPath string, storagePath string) (Result, error) {
	t1 := time.Now()
	planDir := fmt.Sprintf("%v/%v", storagePath, plan.Name)
	var archive, log string
	var err error
	var filePostFix = formatTimeForFilePostFix(t1.UTC());
	k8Client := getKubernetesClient()

	switch plan.Type {
	case "mongo":
		archive, log, err = jobs.RunMongoBackup(plan, k8Client, tmpPath, filePostFix)
	case "solr":
		archive, log, err = jobs.RunSolrBackup(plan, tmpPath, filePostFix)
	case "file":
		archive, log, err = jobs.RunFileBackup(plan, tmpPath, filePostFix)
	}

	res := Result{
		Plan:      plan.Name,
		Timestamp: t1.UTC(),
		Status:    500,
	}
	_, res.Name = filepath.Split(archive)

	if err != nil {
		return res, err
	}

	err = sh.Command("mkdir", "-p", planDir).Run()
	if err != nil {
		return res, errors.Wrapf(err, "creating dir %v in %v failed", plan.Name, storagePath)
	}

	fi, err := os.Stat(archive)
	if err != nil {
		return res, errors.Wrapf(err, "stat file %v failed", archive)
	}
	res.Size = fi.Size()

	err = sh.Command("mv", archive, planDir).Run()
	if err != nil {
		return res, errors.Wrapf(err, "moving file from %v to %v failed", archive, planDir)
	}

	err = sh.Command("mv", log, planDir).Run()
	if err != nil {
		logrus.WithField("file", log).WithField("target directory", planDir).Warn("failed to move log file")
	}

	file := filepath.Join(planDir, res.Name)

	if plan.SFTP != nil {
		sftpOutput, err := sftpUpload(file, plan)
		if err != nil {
			return res, err
		} else {
			logrus.WithField("plan", plan.Name).Info(sftpOutput)
		}
	}

	if plan.S3 != nil {
		s3Output, err := s3Upload(file, plan)
		if err != nil {
			return res, err
		} else {
			logrus.WithField("plan", plan.Name).Infof("S3 upload finished %v", s3Output)
		}
	}

	if plan.GCloud != nil {
		gCloudOutput, err := gCloudUpload(file, plan)
		if err != nil {
			return res, err
		} else {
			logrus.WithField("plan", plan.Name).Infof("GCloud upload finished %v", gCloudOutput)
		}
	}

	if plan.Azure != nil {
		azureOutout, err := azureUpload(file, plan)
		if err != nil {
			return res, err
		} else {
			logrus.WithField("plan", plan.Name).Infof("Azure upload finished %v", azureOutout)
		}
	}

	if plan.Scheduler.Retention > -1 {
		err = applyRetention(planDir, plan.Scheduler.Retention, plan.Scheduler.LogRetention)
		if err != nil {
			return res, errors.Wrap(err, "retention job failed")
		}
	}

	t2 := time.Now()
	res.Status = 200
	res.Duration = t2.Sub(t1)
	return res, nil
}

func formatTimeForFilePostFix(t time.Time) string {
	return t.Format("20060102030405000")
}

func getKubernetesClient() *kubernetes.Clientset {
	var kflags *string
	var err error
	var kubeconfig *rest.Config
	if (config.AppEnv == "production") {
		// creates the in-cluster config
		kubeconfig, err = rest.InClusterConfig()
	} else {

		if home := homeDir(); home != "" {
			kflags = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kflags = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}

		flag.Parse()

		// use the current context in kubeconfig
		kubeconfig, err = clientcmd.BuildConfigFromFlags("", *kflags)
	}


	if err != nil {
		panic(err.Error())
	}

	// creates the clientset
	k8Client, err := kubernetes.NewForConfig(kubeconfig)

	if err != nil {
		panic(err.Error())
	}

	return k8Client
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
