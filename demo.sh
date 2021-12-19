# a quick, ready-to-use demo
echo "*** Distgrep demo ***"
echo -E "" 
if ! [ -f "go.mod" ]; then 
    echo "Please run ./setup.sh first"
    exit -1
fi
echo "This script will run a demo of the 'distgrep' program"
echo "The following entities will be run:"
echo "- distgrep CLIENT, in a new window"
echo "- distgrep SERVER, in this window"
echo "- three distgrep WORKERs, in three new separated window"
echo "Each entity will be identified by the terminal window title."
echo -E ""
read -n 1 -s -r -p "Press any key to start..."
echo -E ""
echo -E "-------------------------------------"

# detect default terminal
terms=($TERMINAL x-terminal-emulator urxvt rxvt termit terminator xterm xfce4-terminal xdg-terminal gnome-terminal iterm)
for t in ${terms[*]}
do
    if [ $(command -v $t) ]
    then
        detected_term=$t
        break
    fi
done

# run workers
base_addr_space=40042
workersno=3
for i in $(seq 1 $workersno)
do
    $detected_term -T "WORKER n. $i"    -e "bash -c './worker/bin/worker -p $(($base_addr_space + $i));bash'"    
done
# run client
$detected_term -T "CLIENT" -e "bash -c 'echo -E In order to run the client please run:;echo -E ./client;echo -E or see:;echo -E ./client -help;echo -E for all the available flags;cd client/bin;bash'"

# run server
cd server
./bin/server
