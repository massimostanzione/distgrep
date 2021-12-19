// DistGrep client
package main

import (
	"context"
	pb "distgrep/client/intf"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

var HighlightType = map[string]uint8{
	"none":      0,
	"classic":   1,
	"asterisks": 2,
}

func main() {
	// Parsing input arguments
	filepath := flag.String("f", "../../ILIAD_1STBOOK_IT_ALTERED", "source file to be \"distgrepp-ed\"")
	substr := flag.String("substr", "Achille", "substr to be searched into the source file")
	serverAddr := flag.String("s", "localhost:40042", "server address and port, in the format ADDRESS:PORT")
	highlight := flag.String("hl", "classic", "[classic/asterisks/none] set substr highlighting in the output\nNOTICE: \"classic\" option may be not available on all systems.")
	help := flag.Bool("help", false, "show this message")

	flag.Parse()
	_, exists := HighlightType[*highlight]
	if !exists {
		fmt.Println("\"-hl\" flag not correctly set.\nSee 'distgrep -help' for allowed values.")
		os.Exit(-1)
	}
	if *help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Contacting the server
	conn, err := grpc.Dial(*serverAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Error while contacting server on %v:\n %v", *serverAddr, err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Defining client interface, to be used to invoke the distgrep service
	cs := pb.NewDistGrepClient(conn)

	// Reading input file
	file, err := ioutil.ReadFile(*filepath)
	if err != nil {
		log.Fatalf("Error while reading file %v:\n%v", *filepath, err)
	}
	content := string(file) // Input file will be passed to server as an array of strings

	// Invoking the service
	resp, err := cs.ExecuteDistGrep(ctx, &pb.DistGrepInput{File: content, Substr: *substr})
	if err != nil {
		log.Fatalf("Error while executing distgrep:\n%v", err)
	}

	// Just a bit of output formatting
	colorReset := "\033[0m"
	colorRed := "\033[31m"
	bold := "\033[1m"
	switch *highlight {
	case "classic":
		resp.Res = strings.Replace(resp.Res, *substr, bold+colorRed+*substr+colorReset, -1)
	case "asterisks":
		resp.Res = strings.Replace(resp.Res, *substr, "*"+*substr+"*", -1)
	case "none":
	default:
	}
	fmt.Printf("Freq.\tLine\n-----------------------------------------\n%v", resp.Res)
}
