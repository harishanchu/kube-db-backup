package jobs

import (
	"github.com/harishanchu/kube-db-backup/config"
	"fmt"
	"github.com/sendgrid/go-solr"
	"net"
	"github.com/codeskyblue/go-sh"
	"net/http"
	"io/ioutil"
	"encoding/json"
)

type PodToBackup struct {
	BaseURL       string
	podName       string
	podNameSpace  string
	BackupSuccess bool
}

type SolrBackupResponse struct {
	Status string `json:"status"`
	ResponseHeader struct {
		Status int `json:"status"`
		QTime  int `json:"QTime"`
	} `json:"response"`
	Exception string `json:"exception"`
}

type SolrReplicaDetailsResponse struct {
	ResponseHeader struct {
		Status int `json:"status"`
		QTime  int `json:"QTime"`
	} `json:"response"`
	Details map[string]interface{
	} `json:"details"`
	Exception string `json:"exception"`
}

func RunSolrBackup(plan config.Plan, tmpPath string, filePostFix string) (string, string, error) {
	archive := fmt.Sprintf("%v/%v-%v.gz", tmpPath, plan.Name, filePostFix)
	log := fmt.Sprintf("%v/%v-%v.log", tmpPath, plan.Name, filePostFix)

	zkHost := plan.Target["zkHost"];
	solrCollection := plan.Target["collection"]

	solrZk, err := CreateSolrCloudConnection(zkHost, solrCollection)

	podsToBackup := make([]PodToBackup, 0)

	if err != nil {

	} else {
		state, err := solrZk.GetClusterState()
		state = solr.ClusterState{
			make([]string, 0),
			1,
			map[string]solr.Collection{
				"mpi": solr.Collection{
					map[string]solr.Shard{
						"shard1": solr.Shard{
							"test",
							"sddsf",
							"active",
							map[string]solr.Replica{
								"replica1": solr.Replica{
									"hello",
									"true",
									"http://localhost:8983/solr",
									"testurl",
									"active",
								},
							},
						},
					},
					"123",
				},
			},
		}

		if err != nil {

		} else {
			podsToBackup = RetrievePodsToBackup(state.Collections[solrCollection].Shards)
		}
	}

	httpClient := &http.Client{}

	for _, pod := range podsToBackup {
		resp, err := InitiateReplicaBackup(httpClient, pod.BaseURL, solrCollection)

		if (err != nil || resp.Exception != "") {
		}
	}

	for len(podsToBackup) > 0 {
		for index, pod := range podsToBackup {
			resp, err := CheckReplicaBackupStatus(httpClient, pod.BaseURL, solrCollection)

			if (err != nil || resp.Exception != "") {
			} else {
				//backupStatus := reflect.ValueOf(resp.Details["backup"])
				//e := backupStatus.Index(5).Elem()
				//fmt.Print(reflect.TypeOf(e))
				//if backupStatus.Index(5).String() == "success" {
					podsToBackup = append(podsToBackup[:index], podsToBackup[index+1:]...)
				//} else {
				//	fmt.Print(backupStatus.Index(5).String())
				//}
			}
		}
	}

	return archive, log, nil
}

func CreateSolrCloudConnection(host, collection string) (solr.SolrZK, error) {
	var solrzk = solr.NewSolrZK(host, "", collection)
	var err = solrzk.Listen()

	if err != nil {
		return nil, err
	} else {
		return solrzk, nil
	}
}

func RetrievePodsToBackup(shards map[string]solr.Shard) []PodToBackup {
	podsToBackup := make([]PodToBackup, 0)

	for _, shard := range shards {
		if shard.State == "active" {
			for _, replica := range shard.Replicas {
				if replica.Leader == "true" && replica.State == "active" {
					podIps, _ := net.LookupIP(replica.NodeName)

					if len(podIps) > 0 {
						podNameCommand := fmt.Sprintf("kubectl get pod --all-namespaces -o "+
							"jsonpath='{range .items[*]}{.metadata.name} {..podIP} "+
							"{.status.containerStatuses[0].state}{\"\\n\"}{end}' "+
							"--sort-by=.metadata.name|grep running|grep %v|awk '{printf $1}'", podIps[0])
						podNameSpaceCommand := fmt.Sprintf("kubectl get pod --all-namespaces -o "+
							"jsonpath='{range .items[*]}{.metadata.namespace} {..podIP} "+
							"{.status.containerStatuses[0].state}{\"\\n\"}{end}' "+
							"--sort-by=.metadata.namespace|grep running|grep %v|awk '{printf $1}'", podIps[0])

						podName, err1 := sh.Command("/bin/sh", "-c", podNameCommand).Output()
						podNameSpace, err2 := sh.Command("/bin/sh", "-c", podNameSpaceCommand).Output()

						if err1 == nil && err2 == nil {
							podToBackup := PodToBackup{
								replica.BaseURL,
								string(podName),
								string(podNameSpace),
								false,
							}

							podsToBackup = append(podsToBackup, podToBackup)
							break;
						}
					}
				}
			}
		}
	}

	return podsToBackup
}

func InitiateReplicaBackup(httpClient *http.Client, nodeUri string, collection string) (SolrBackupResponse, error) {
	backupUrl := fmt.Sprintf("%s/%s/replication?command=backup&wt=json", nodeUri, collection)
	req, err := http.NewRequest("GET", backupUrl, nil)
	var sr SolrBackupResponse

	if err != nil {
		return sr, err
	}

	req.Header.Add("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		htmlData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return sr, err
		}

		if resp.StatusCode < 500 {
			return sr, solr.NewSolrError(resp.StatusCode, string(htmlData))
		} else {
			return sr, solr.NewSolrInternalError(resp.StatusCode, string(htmlData))
		}
	}

	dec := json.NewDecoder(resp.Body)

	return sr, dec.Decode(&sr)
}

func CheckReplicaBackupStatus (httpClient *http.Client, nodeUri string, collection string) (SolrReplicaDetailsResponse, error) {
	replicaDetailsUrl := fmt.Sprintf("%s/%s/replication?command=details&wt=json", nodeUri, collection)
	req, err := http.NewRequest("GET", replicaDetailsUrl, nil)
	var sr SolrReplicaDetailsResponse

	if err != nil {
		return sr, err
	}

	resp, err := httpClient.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		htmlData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return sr, err
		}

		if resp.StatusCode < 500 {
			return sr, solr.NewSolrError(resp.StatusCode, string(htmlData))
		} else {
			return sr, solr.NewSolrInternalError(resp.StatusCode, string(htmlData))
		}
	}

	dec := json.NewDecoder(resp.Body)

	return sr, dec.Decode(&sr)
}