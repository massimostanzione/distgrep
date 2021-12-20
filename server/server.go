// DistGrep server
package main

import (
	"bufio"
	"context"
	pb "distgrep/server/intf"
	pbw "distgrep/worker/intf"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	MIN_WORKERSNO = 1
	MAX_WORKERSNO = 15
)

var workersArray []string

type DGserver struct {
	pb.UnimplementedDistGrepServer
}

type DistributedSliceIndexes struct {
	workerAddr string
	from       int
	to         int
}

// Auxiliary function to assign lines from the files into an array of strings
func splitLines(s string) []string {
	var lines []string
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}

	return lines
}

// "Map" assignation: called concurrently, it calls a single worker (as mapper) and make this server act as a client towards it.
func mapAssign(ctxw context.Context, wAddr string, lines []string, substr string, responses chan (map[string]uint32)) {
	wConn, err := grpc.Dial(wAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Error while contacting worker on %v:\n %v", wAddr, err)
	}
	defer wConn.Close()
	w := pbw.NewDistGrepWorkerClient(wConn)
	log.Printf("- Launching Map at %v", wAddr)
	resp, err := w.Map(ctxw, &pbw.MapInput{File: lines, Substr: substr})
	if err != nil {
		log.Fatalf("Error while executing map on %v:\n%v", wAddr, err)
	}
	responses <- resp.Res
}

// "Reduce" assignation: called concurrently, it calls a single worker (as reducer) and make this server act as a client towards it.
func reduceAssign(ctxw context.Context, wAddr string, lines []map[string]*pbw.ReduceInput_OccurrenceList, substr string, responses chan (map[string]uint32)) {
	wConn, err := grpc.Dial(wAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Error while contacting worker on %v:\n %v", wAddr, err)
	}
	defer wConn.Close()
	w := pbw.NewDistGrepWorkerClient(wConn)

	// Worker needs to receive a "real" map, while at this point the input is still an array (see calling code)
	mappa := make(map[string]*pbw.ReduceInput_OccurrenceList)
	for i := range lines {
		app := lines[i]
		for k, v := range app {
			mappa[k] = v
			app = nil
		}
	}

	log.Printf("- Launching Reduce at %v", wAddr)
	resp, err := w.Reduce(ctxw, &pbw.ReduceInput{Res: mappa})
	if err != nil {
		log.Fatalf("Error while executing map on %v:\n%v", wAddr, err)
	}
	responses <- resp.Res
}

// Auxiliary function: computation of the indexes of the slices of input to be passed to workers
func computeSlices(length int) []DistributedSliceIndexes {
	log.Println("- Workload distribution:")
	log.Println("  From\tTo\tAddress")
	var rapp float64
	slicesIndexes := make([]DistributedSliceIndexes, 0)
	rapp = float64(float64(length) / float64(len(workersArray)))
	log.Println("  ----------------------------")
	stop := false
	for i := range workersArray {
		slicesIndex := DistributedSliceIndexes{}
		from := int(float64(i) * math.Ceil(rapp))
		to := int(float64(i+1) * math.Ceil(rapp))
		if from >= length {
			from = length
			stop = true
		}
		if to >= length {
			to = length
		}
		if stop {
			break
		}
		log.Printf("  %-2v\t%v\t%v", from, to-1, workersArray[i]) //"to" not included, print the effective index
		slicesIndex.workerAddr = workersArray[i]
		slicesIndex.from = from
		slicesIndex.to = to
		slicesIndexes = append(slicesIndexes, slicesIndex)
	}
	return slicesIndexes
}

// Synchronization point: here we read the output arriving from the workers, into the specified channel,
// and we know that exactly len(workersArray) maps have to be received from them.
func syncPointRead(channel chan map[string]uint32, effectiveWorkersNo int) []map[string]uint32 {
	var ret = make([]map[string]uint32, len(workersArray))
	for i := 0; i < effectiveWorkersNo; i++ {
		resp := <-channel
		ret[i] = resp
	}
	return ret
}

// The main flow of DistGrep execution.
func (s *DGserver) ExecuteDistGrep(ctx context.Context, in *pb.DistGrepInput) (*pb.DistGrepOutput, error) {
	log.Printf("*** REQUEST RECEIVED ***")
	log.Printf("Request data:")
	lines := splitLines(in.File)
	log.Printf("- Source\t[%v lines, omitted]", len(lines))
	log.Printf("- Substr\t\"%v\"", in.Substr)
	ctxw, _ := context.WithTimeout(context.Background(), time.Second)

	// MAP
	log.Printf("Map phase")
	responses := make(chan map[string]uint32)
	defer close(responses)
	var val uint32
	val = 0
	var mapped []map[string]uint32
	var slicesIndexes []DistributedSliceIndexes
	if len(lines) == 0 {
		log.Printf("- Nothing to Map: input is empty")
	} else {
		slicesIndexes := computeSlices(len(lines))
		for i := range slicesIndexes {
			from := slicesIndexes[i].from
			to := slicesIndexes[i].to
			go mapAssign(ctxw, workersArray[i], lines[from:to], in.Substr, responses)
		}
		// Read from mappers (synchronization point)
		mapped = syncPointRead(responses, len(slicesIndexes))

		// Just for printing and verifying
		for i := range mapped {
			for _, v := range mapped[i] {
				val += v
			}
		}
		log.Printf("- %v line(s) matched the substr.", val)
	}
	log.Printf("Map completed.")

	// SHUFFLE & SORT
	// NOTICE: since the "map" native structure does not allow multiple values for keys, the pairs received have
	//         as value the sum of the occurrences of the key, instead of "1"
	log.Printf("Shuffle & Sort phase")
	sns := make(map[string][]uint32)
	if val == 0 {
		log.Printf("- Nothing to shuffle&sort: none of the input strings matched with the substring")
	} else {
		for i := range mapped {
			for k, v := range mapped[i] {
				sns[k] = append(sns[k], v)
			}
		}
		log.Printf("- %v unique line(s) matched the substr.", len(sns))
	}
	log.Printf("Shuffle & Sort completed.")

	// REDUCE
	log.Printf("Reduce phase")
	reduced := make(map[string]uint32)
	if len(sns) == 0 {
		log.Printf("- Nothing to reduce: none of the input strings matched with the substring")
		reduced = nil
	} else {
		// Since the "map" native structure is not sorted, and each iteration is done randomly, we need a
		// temporary array to be iterated and sliced to be sent to the reducers
		mapArray := make([]map[string]*pbw.ReduceInput_OccurrenceList, 0)
		for k, v := range sns {
			mapItem := make(map[string]*pbw.ReduceInput_OccurrenceList)
			l := &pbw.ReduceInput_OccurrenceList{Occurrence: v}
			mapItem[k] = l
			mapArray = append(mapArray, mapItem)
			mapItem = nil
			l = nil
		}
		// Re-distributing the mapped workload among the same workers, that act now as reducers
		slicesIndexes = nil
		slicesIndexes = computeSlices(len(sns))
		for i := range slicesIndexes {
			from := slicesIndexes[i].from
			to := slicesIndexes[i].to
			go reduceAssign(ctxw, workersArray[i], mapArray[from:to], in.Substr, responses)
		}

		// Read from mappers (synchronization point)
		mapped = syncPointRead(responses, len(slicesIndexes))
		for i := range mapped {
			for k, v := range mapped[i] {
				reduced[k] = v
			}
		}
	}
	// Preparing output
	ress := ""
	for k, v := range reduced {
		ress += strconv.FormatUint(uint64(v), 10)
		ress += "\t"
		ress += string(k)
		ress += "\n"
	}

	log.Println("Reduce completed.")
	log.Println("Done. Listening for new requests...")
	return &pb.DistGrepOutput{Res: ress}, status.New(codes.OK, "").Err()
}

func main() {
	log.Println("*** DISTGREP SERVER ***")
	// Parsing input arguments
	port := flag.String("p", "40042", "port to listen for distgrep requests")
	workers := flag.String("w", "localhost:40043;localhost:40044;localhost:40045", "addresses and ports of the workers to be bound with, in the following format:\nADDRESS_1:PORT_1;ADDRESS2:PORT_2;...;ADDRESS_N:PORT_N\nMust be between 1 and 15")
	help := flag.Bool("help", false, "show this message")

	flag.Parse()
	if *help {
		flag.PrintDefaults()
		os.Exit(0)
	}
	workersArray = strings.Split(*workers, ";")
	if len(workersArray) < MIN_WORKERSNO || len(workersArray) > MAX_WORKERSNO {
		fmt.Printf("Workers to be bound with must be between %v and %v\n", MIN_WORKERSNO, MAX_WORKERSNO)
		os.Exit(-1)
	}
	log.Println("Will bind (on-demand) to the following workers:")
	for i := range workersArray {
		log.Printf("- %v", workersArray[i])
	}
	// Start listening for incoming calls
	lis, err := net.Listen("tcp", "localhost:"+*port)
	if err != nil {
		log.Fatalf("Error while trying to listen to port %v:\n%v", *port, err)
	}
	log.Printf("--------------------------")
	log.Printf("Listening on port %v...", *port)
	// New server instance and service registering
	s := grpc.NewServer()
	pb.RegisterDistGrepServer(s, &DGserver{})
	// Serve incoming calls
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Error while trying to serve request: %v", err)
	}
}
