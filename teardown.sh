#!/bin/bash
# run this script if something goes wrong
echo -E "Cleaning up..."
sudo rm -f go.mod go.sum
	        
for item in client server worker
do
    sudo rm -f $item/go.mod \
	           $item/go.sum \
	           $item/intf/*.go \
	           $item/bin/$item
done
echo -E "Done."
