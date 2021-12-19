// DistGrep worker
package main

import (
	"context"
	pb "distgrep/worker/intf"
	"flag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"net"
	"os"
	"strings"
)

type DGworker struct {
	pb.UnimplementedDistGrepWorkerServer
}

// The Map function.
// NOTICE: since the "map" native structure does not allow multiple values for keys, the pairs received
//         already have as value the sum of the occurrences of the key, instead of "1"
func (s *DGworker) Map(ctx context.Context, in *pb.MapInput) (*pb.MapReduceOutput, error) {
	log.Printf("*** REQUEST RECEIVED ***")
	log.Printf("- Type:\tMAP")
	log.Printf("- Source:\t[%v lines, omitted]", len(in.File))
	log.Printf("- Substr:\t\"%v\"", in.Substr)

	log.Printf("Starting Map...")
	var occ = make(map[string]uint32)
	for i := 0; i < len(in.File); i++ {
		occurrencesNo := strings.Count(in.File[i], in.Substr)
		if occurrencesNo > 0 {
			// NOTICE: since the "map" native structure does not allow multiple values for keys, the pairs received
			//         already have as value the sum of the occurrences of the key, instead of "1"
			occ[in.File[i]] = occ[in.File[i]] + 1
		}
	}
	log.Printf("- %v match(es) found.", len(occ))
	log.Println("Done. Listening for new requests...")
	return &pb.MapReduceOutput{Res: occ}, status.New(codes.OK, "").Err()
}

// The Reduce function.
func (s *DGworker) Reduce(ctx context.Context, in *pb.ReduceInput) (*pb.MapReduceOutput, error) {
	log.Printf("*** REQUEST RECEIVED ***")
	log.Printf("- Type:\tREDUCE")
	log.Printf("- Source:\t[%v lines, omitted]", len(in.Res))

	log.Printf("Starting Reduce...")

	ret := make(map[string]uint32)
	for k, v := range in.Res {
		for j := range v.Occurrence {
			ret[k] += uint32(v.Occurrence[j])
		}
	}

	log.Println("Done. Listening for new requests...")
	return &pb.MapReduceOutput{Res: ret}, status.New(codes.OK, "").Err()
}

func main() {
	log.Println("*** DISTGREP WORKER ***")
	// Parsing input arguments
	port := flag.String("p", "40042", "port to listen for distgrep requests")
	help := flag.Bool("help", false, "show this message")

	flag.Parse()
	if *help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Start listening for incoming calls
	lis, err := net.Listen("tcp", "localhost:"+*port)
	if err != nil {
		log.Fatalf("Error while trying to listen to port %v:\n%v", *port, err)
	}
	log.Printf("--------------------------")
	log.Printf("Listening on port %v...", *port)
	// New server instance and service registering
	w := grpc.NewServer()
	pb.RegisterDistGrepWorkerServer(w, &DGworker{})
	// Serve incoming calls
	if err := w.Serve(lis); err != nil {
		log.Fatalf("Error while trying to serve request: %v", err)
	}
}
