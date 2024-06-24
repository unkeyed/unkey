package cluster


type Cluster interface {
	 Join(addrs []string) (clusterSize int, err error) 
	 Shutdown() error
	  FindNodes(key string, n int) ([]Node, error) 

	}