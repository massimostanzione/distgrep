# distgrep
*A distributed `grep`, implemented with the **MapReduce** model.*

This repository consist of an exercise made in the context of the *Sistemi Distribuiti e Cloud Computing* (SDCC) course, taught at Faculty of Engineering, University of Rome Tor Vergata.
#### The assignment
Realize a distributed `grep` using the ***MapReduce*** paradigm: DistGrep returns the lines of text of a large input file given in input that match a specific pattern (i.e.,regular expression) specified. Requirements: use Go and RPC (or gRPC)

#### How does it work
The program consists of three entities: **client**, **server**, **worker**.
Workers can play the role of mappers and reducers.
- A **client** entity sends a `distgrep(file, substr)` request to a **server** entity, with a source `file` and a `substr` to search within it
- The **server** splits the `file` in `n` blocks, and assigns it to `n` **worker**s with a `map(block, substr)` request
- **(Map)** The **worker** entities execute the ***Map*** function, and emit a sequence of key-value pairs `(k,v)` where the key `k` is a single line of the received block matching the `substr`, and `v` is `"1"` *(actual implementation: the native `map` structure does not allow multiple values, so `v` is actually the number of occurrence of the line `k` into the block sent to the single worker)*
- **(Shuffle and Sort)** The **server** receives the pairs from the **worker**s and re-arranges them in such a way as to obtain a new sequence of key-value pairs `(k,v')` where `v'` is now a sequence of all the occurrences of `k` into all the blocks processed by the *mappers*.
- **(Reduce)** The **server** splits again these pairs among the **worker**s, that now act as *reducers*, with a `reduce(pairs)` request. They will return a final sequence of key-value pairs `(k,v'')`, where `v''` is now the number of occurrences of `k` into the whole `file`.
- The **server** returns back the pairs, re-arranged as a string, to the **client**, that will print it.

#### Communication
Communication is implemented via gRPC, with the interfaces declared with Protocol Buffer as IDL.

#### How to run it
1. Run `setup.sh` to create the gRPC stubs, arrange all the dependencies and generate executables.
2. Then you can either run individually the single entities (the executables generated in the `/<entity>/bin` folder or `go run main.go <params>`) or execute a ready-to-use demonstration by running `demo.sh`.

The input file provided, `ILIAD_1STBOOK_IT_ALTERED`, is the italian translation of the first book of the [Homer](http://https://en.wikipedia.org/wiki/Homer "Homer")'s [*Iliad*](https://en.wikipedia.org/wiki/Iliad/ "Iliad"), but altered in such a way that some lines are randomly repeated across the whole file.

For each entity, please refer to the `-help` flag to see all the options available.

If something goes wrong, please run `teardown.sh` to reset the environment.

#### Notice
- **Final output**: it is slightly different from a typical `grep` output:
	- Since with **MapReduce** we work on the occurrences of the lines, they are **both** printed in output.
	- Since the `map` native Go structure is not sorted in any way, the output has a **different sorting at each run**.

Realized with vim-go, xed, GoClipse.
