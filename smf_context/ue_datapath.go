package smf_context

import (
	"fmt"
)

type UEDataPathGraph struct {
	SUPI  string
	Graph []*DataPathNode
}

func NewUEDataPathNode(name string) (node *DataPathNode, err error) {

	upNodes := smfContext.UserPlaneInformation.UPNodes

	if _, exist := upNodes[name]; !exist {
		err = fmt.Errorf("UPNode %s isn't exist in smfcfg.conf, but in UERouting.yaml!", name)
		return nil, err
	}

	node = &DataPathNode{
		UPF:              upNodes[name].UPF,
		Next:             make(map[string]*DataPathLink),
		Prev:             nil,
		IsBranchingPoint: false,
	}
	return
}

func NewUEDataPathGraph(SUPI string) (UEPGraph *UEDataPathGraph, err error) {

	UEPGraph = new(UEDataPathGraph)
	UEPGraph.Graph = make([]*DataPathNode, 0)
	UEPGraph.SUPI = SUPI

	paths := smfContext.UERoutingPaths[SUPI]
	lowerBound := 0

	NodeCreated := make(map[string]*DataPathNode)

	for _, path := range paths {
		upperBound := len(path.UPF) - 1

		DataEndPoint := &DataPathLink{
			DestinationIP:   path.DestinationIP,
			DestinationPort: path.DestinationPort,
		}
		for idx, node_name := range path.UPF {

			var ue_node, child_node, parent_node *DataPathNode
			var exist bool
			var err error

			if ue_node, exist = NodeCreated[node_name]; !exist {

				ue_node, err = NewUEDataPathNode(node_name)

				if err != nil {
					return nil, err
				}
				NodeCreated[node_name] = ue_node
				UEPGraph.Graph = append(UEPGraph.Graph, ue_node)
			}

			switch idx {
			case lowerBound:
				child_name := path.UPF[idx+1]

				if child_node, exist = NodeCreated[child_name]; !exist {
					child_node, err = NewUEDataPathNode(child_name)

					if err != nil {
						return nil, err
					}
					NodeCreated[child_name] = child_node
					UEPGraph.Graph = append(UEPGraph.Graph, child_node)
				}

				//fmt.Printf("%+v\n", ue_node)
				ue_node.AddChild(child_node)
				ue_node.AddDestinationOfChild(child_node, DataEndPoint)

			case upperBound:
				parent_name := path.UPF[idx-1]

				if parent_node, exist = NodeCreated[parent_name]; !exist {
					parent_node, err = NewUEDataPathNode(parent_name)

					if err != nil {
						return nil, err
					}
					NodeCreated[parent_name] = parent_node
					UEPGraph.Graph = append(UEPGraph.Graph, parent_node)
				}

				//fmt.Printf("%+v\n", ue_node)
				ue_node.AddParent(parent_node)
			default:
				child_name := path.UPF[idx+1]

				if child_node, exist = NodeCreated[child_name]; !exist {
					child_node, err = NewUEDataPathNode(child_name)

					if err != nil {
						return nil, err
					}
					NodeCreated[child_name] = child_node
					UEPGraph.Graph = append(UEPGraph.Graph, child_node)
				}

				parent_name := path.UPF[idx-1]

				if parent_node, exist = NodeCreated[parent_name]; !exist {
					parent_node, err = NewUEDataPathNode(parent_name)

					if err != nil {
						return nil, err
					}
					NodeCreated[parent_name] = parent_node
					UEPGraph.Graph = append(UEPGraph.Graph, parent_node)
				}

				//fmt.Printf("%+v\n", ue_node)
				ue_node.AddChild(child_node)
				ue_node.AddDestinationOfChild(child_node, DataEndPoint)
				ue_node.AddParent(parent_node)
			}

		}
	}

	return
}

func (uepg *UEDataPathGraph) PrintGraph() {

	fmt.Println("SUPI: ", uepg.SUPI)
	upi := smfContext.UserPlaneInformation

	for _, node := range uepg.Graph {
		fmt.Println("\tUPF Name: ")
		node_ip := node.GetNodeIP()
		fmt.Println("\t\t", upi.GetUPFNameByIp(node_ip))

		fmt.Println("\tBranching Point: ")
		fmt.Println("\t\t", node.IsBranchingPoint)

		if node.Prev != nil {
			fmt.Println("\tParent Name: ")
			parent_ip := node.Prev.To.GetNodeIP()
			fmt.Println("\t\t", upi.GetUPFNameByIp(parent_ip))
		}

		if node.Next != nil {
			fmt.Println("\tChildren Name: ")
			for _, child_link := range node.Next {

				child_ip := child_link.To.GetNodeIP()
				fmt.Println("\t\t", upi.GetUPFNameByIp(child_ip))
				fmt.Println("\t\tDestination IP: ", child_link.DestinationIP)
				fmt.Println("\t\tDestination Port: ", child_link.DestinationPort)
			}
		}
	}
}

func (uepg *UEDataPathGraph) FindBranchingPoints() {
	//BFS algo implementation
	const (
		WHITE int = 0
		GREY  int = 1
		BLACK int = 2
	)

	num_of_nodes := len(uepg.Graph)

	color := make(map[string]int)
	distance := make(map[string]int)
	queue := make(chan *DataPathNode, num_of_nodes)

	for _, node := range uepg.Graph {

		node_id, _ := node.GetUPFID()
		color[node_id] = WHITE
		distance[node_id] = num_of_nodes + 1
	}

	cur_idx := 0 // start point
	for j := 0; j < num_of_nodes; j++ {

		node_id, _ := uepg.Graph[cur_idx].GetUPFID()
		if color[node_id] == WHITE {
			color[node_id] = GREY
			distance[node_id] = 0

			queue <- uepg.Graph[cur_idx]
			for len(queue) > 0 {
				node := <-queue
				branchingCount := 0
				for child_id, child_link := range node.Next {

					if color[child_id] == WHITE {
						color[child_id] = GREY
						distance[child_id] = distance[node_id] + 1
						queue <- child_link.To
					}

					if color[child_id] == WHITE || color[child_id] == GREY {
						branchingCount += 1
					}
				}

				if node.Prev != nil {

					parent := node.Prev.To
					parent_id, _ := node.Prev.To.GetUPFID()

					if color[parent_id] == WHITE {
						color[parent_id] = GREY
						distance[parent_id] = distance[node_id] + 1
						queue <- parent
					}

					if color[parent_id] == WHITE || color[parent_id] == GREY {
						branchingCount += 1
					}
				}

				if branchingCount >= 2 {
					node.IsBranchingPoint = true
				}
				color[node_id] = BLACK
			}
		}

		//Keep finding other connected components
		cur_idx = j
	}

}

func (uepg *UEDataPathGraph) GetGraphRoot() *DataPathNode {

	return uepg.Graph[0]
}

func CheckUEHasPreConfig(SUPI string) (exist bool) {

	_, exist = smfContext.UERoutingGraphs[SUPI]
	return
}

func GetUERoutingGraph(SUPI string) *UEDataPathGraph {

	return smfContext.UERoutingGraphs[SUPI]
}

func ConstructUserPlaneTopoByPath(upPath []*UPNode) (root *DataPathNode) {

	var lowerBound = 0
	var upperBound = len(upPath) - 1

	for idx, node := range upPath {

		switch idx {
		case lowerBound:

		case upperBound:

		default:

		}
	}

}
