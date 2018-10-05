package jobs

import (
	"github.com/harishanchu/kube-db-backup/config"
	"fmt"
	"github.com/sendgrid/go-solr"
	"net"
	"github.com/codeskyblue/go-sh"
)

type PodToBackup struct {
	BaseURL      string
	podName      string
	podNameSpace string
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
			make([]string,0),
			1,
			map[string]solr.Collection {
				"mpi": solr.Collection{
					map[string] solr.Shard {
						"shard1": solr.Shard{
							"test",
							"sddsf",
							"active",
							map[string] solr.Replica {
								"replica1": solr.Replica {
									"hello",
									"true",
									"testurl",
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

	print(podsToBackup)

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
