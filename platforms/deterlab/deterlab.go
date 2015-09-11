// deploy2deter is responsible for kicking off the deployment process
// for deterlab. Given a list of hostnames, it will create an overlay
// tree topology, using all but the last node. It will create multiple
// nodes per server and run timestamping processes. The last node is
// reserved for the logging server, which is forwarded to localhost:8081
//
// options are "bf" which specifies the branching factor
//
// 	and "hpn" which specifies the replicaiton factor: hosts per node
//
// Creates the following directory structure in remote:
// exec, timeclient, logserver/...,
// this way it can rsync the remove to each of the destinations
package platform_deter

import (
/*
"encoding/json"
"fmt"
"io/ioutil"
"strconv"
"strings"
"github.com/ineiti/cothorities/helpers/config"
"github.com/ineiti/cothorities/helpers/graphs"
*/
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/ineiti/cothorities/helpers/cliutils"
	dbg "github.com/ineiti/cothorities/helpers/debug_lvl"
	"github.com/ineiti/cothorities/platforms"
	"fmt"
	"strings"
	"io/ioutil"
	"github.com/ineiti/cothorities/helpers/graphs"
	"encoding/json"
	"github.com/ineiti/cothorities/helpers/config"
	"strconv"
)


type Deter struct {
	Config       *platforms.Config
	// The login on the platform
	Login        string
	// The outside host on the platform
	Host         string
	// The name of the internal hosts
	Project      string
	// Directory where everything is copied into
	BuildDir     string
	// Working directory of deterlab
	DeterDir	 string
	// Where the main logging machine resides
	masterLogger string
	// DNS-resolvable names
	phys         []string
	// VLAN-IP names
	virt         []string
	physOut      string
	virtOut      string
}

func (d *Deter) Configure(config *platforms.Config) {
	d.Config = config
	d.Login = "ineiti"
	d.Host = "users.deterlab.net"
	d.Project = "Dissent-CS"
	d.DeterDir = "platform/deterlab"
	pwd, _ := os.Getwd()
	d.BuildDir = pwd + "/" + d.DeterDir + "/build"
	os.MkdirAll(d.BuildDir, 0777)

	d.generateHostsFile()
	d.readHosts()
	d.calculateGraph()
}

func (d *Deter) Build() (error) {
	dbg.Lvl1("Building for", d.Login, d.Host, d.Project)
	var wg sync.WaitGroup

	// Start with a clean build-directory
	current, _ := os.Getwd()
	dbg.Lvl1(current)
	defer os.Chdir(current)

	// Go into deterlab-dir and create the build-dir
	os.Chdir(d.DeterDir)
	os.RemoveAll(d.BuildDir)
	os.Mkdir(d.BuildDir, 0777)
	dbg.Lvl1(os.Getwd())
	dbg.Lvl1(d.DeterDir)
	dbg.Lvl1(d.BuildDir)

	// start building the necessary packages
	dbg.Lvl2("Starting to build all executables")
	packages := []string{"logserver", "timeclient", "forkexec", "exec", "deter"}
	for _, p := range packages {
		dbg.Lvl2("Building ", p)
		wg.Add(1)
		if p == "deter" {
			go func(p string) {
				defer wg.Done()
				// the users node has a 386 FreeBSD architecture
				err := cliutils.Build(p, d.BuildDir + "/" + p, "386", "freebsd")
				if err != nil {
					cliutils.KillGo()
					log.Fatal(err)
				}
			}(p)
			continue
		}
		go func(p string) {
			defer wg.Done()
			// deter has an amd64, linux architecture
			err := cliutils.Build(p, d.BuildDir + "/" + p, "amd64", "linux")
			if err != nil {
				cliutils.KillGo()
				log.Fatal(err)
			}
		}(p)
	}
	// wait for the build to finish
	wg.Wait()
	dbg.Lvl2("Build is finished")

	// copy the webfile-directory of the logserver to the build directory
	err := exec.Command("cp", "-a", "logserver/webfiles", d.BuildDir).Run()
	if err != nil {
		log.Fatal("error copying webfiles directory into building directory:", err)
	}
	dbg.Lvl2("Done building")
	return nil
}

func (d *Deter) Deploy() (error) {
	// Copy everything over to deterlabs
	err := cliutils.Rsync(d.Login, d.Host, d.BuildDir + "/", "remote/")
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func (d *Deter) Start() (error) {
	// setup port forwarding for viewing log server
	// ssh -L 8081:pcXXX:80 username@users.isi.deterlab.net
	// ssh username@users.deterlab.net -L 8118:somenode.experiment.YourClass.isi.deterlab.net:80
	dbg.Lvl2("setup port forwarding for master logger: ", d.masterLogger, d.Login, d.Host)
	cmd := exec.Command(
		"ssh",
		"-t",
		"-t",
		fmt.Sprintf("%s@%s", d.Login, d.Host),
		"-L",
		"8081:" + d.masterLogger + ":10000")
	err := cmd.Start()
	if err != nil {
		log.Fatal("failed to setup portforwarding for logging server")
	}

	dbg.Lvl2("runnning deter with nmsgs:", d.Config.Nmsgs, d.Login, d.Host)
	// run the deter lab boss nodes process
	// it will be responsible for forwarding the files and running the individual
	// timestamping servers
	dbg.Lvl2(cliutils.SshRunStdout(d.Login, d.Host,
		"GOMAXPROCS=8 remote/deter -nmsgs=" + strconv.Itoa(d.Config.Nmsgs) +
		" -hpn=" + strconv.Itoa(d.Config.Hpn) +
		" -bf=" + strconv.Itoa(d.Config.Bf) +
		" -rate=" + strconv.Itoa(d.Config.Rate) +
		" -rounds=" + strconv.Itoa(d.Config.Rounds) +
		" -debug=" + strconv.Itoa(d.Config.Debug) +
		" -failures=" + strconv.Itoa(d.Config.Failures) +
		" -rfail=" + strconv.Itoa(d.Config.RFail) +
		" -ffail=" + strconv.Itoa(d.Config.FFail) +
		" -app=" + d.Config.App +
		" -suite=" + d.Config.Suite))

	return nil
}

func (d *Deter) Stop() (error) {
	killssh := exec.Command("pkill", "-f", "ssh -t -t")
	killssh.Stdout = os.Stdout
	killssh.Stderr = os.Stderr
	err := killssh.Run()
	if err != nil {
		log.Print("Stopping ssh: ", err)
	}

	return nil
}

/*
* Write the hosts.txt file automatically
* from project name and number of servers
 */
func (d *Deter)generateHostsFile() error {
	hosts_file := d.BuildDir + "/hosts.txt"
	num_servers := d.Config.Nmachs + d.Config.Nloggers

	// open and erase file if needed
	if _, err1 := os.Stat(hosts_file); err1 == nil {
		dbg.Lvl3(fmt.Sprintf("Hosts file %s already exists. Erasing ...", hosts_file))
		os.Remove(hosts_file)
	}
	// create the file
	f, err := os.Create(hosts_file)
	if err != nil {
		log.Fatal("Could not create hosts file description: ", hosts_file, " :: ", err)
		return err
	}
	defer f.Close()

	// write the name of the server + \t + IP address
	ip := "10.255.0."
	name := "SAFER.isi.deterlab.net"
	for i := 1; i <= num_servers; i++ {
		f.WriteString(fmt.Sprintf("server-%d.%s.%s\t%s%d\n", i - 1, d.Project, name, ip, i))
	}
	dbg.Lvl3(fmt.Sprintf("Created hosts file description (%d hosts)", num_servers))
	return err

}

// parse the hosts.txt file to create a separate list (and file)
// of physical nodes and virtual nodes. Such that each host on line i, in phys.txt
// corresponds to each host on line i, in virt.txt.
func (d *Deter)readHosts() {
	hosts_file := d.BuildDir + "/hosts.txt"
	nmachs, nloggers := d.Config.Nmachs, d.Config.Nloggers

	physVirt, err := cliutils.ReadLines(hosts_file)
	if err != nil {
		log.Panic("Couldn't find", hosts_file)
	}

	d.phys = make([]string, 0, len(physVirt) / 2)
	d.virt = make([]string, 0, len(physVirt) / 2)
	for i := 0; i < len(physVirt); i += 2 {
		d.phys = append(d.phys, physVirt[i])
		d.virt = append(d.virt, physVirt[i + 1])
	}
	d.phys = d.phys[:nmachs + nloggers]
	d.virt = d.virt[:nmachs + nloggers]
	d.physOut = strings.Join(d.phys, "\n")
	d.virtOut = strings.Join(d.virt, "\n")
	d.masterLogger = d.phys[0]
	// slaveLogger1 := phys[1]
	// slaveLogger2 := phys[2]

	// phys.txt and virt.txt only contain the number of machines that we need
	dbg.Lvl2("Reading phys and virt")
	err = ioutil.WriteFile(d.BuildDir + "/phys.txt", []byte(d.physOut), 0666)
	if err != nil {
		log.Fatal("failed to write physical nodes file", err)
	}

	err = ioutil.WriteFile(d.BuildDir + "/virt.txt", []byte(d.virtOut), 0666)
	if err != nil {
		log.Fatal("failed to write virtual nodes file", err)
	}
}

// Calculates a tree that is used for the timestampers
func (d *Deter)calculateGraph() {
	d.virt = d.virt[3:]
	d.phys = d.phys[3:]
	t, hostnames, depth, err := graphs.TreeFromList(d.virt, d.Config.Hpn, d.Config.Bf)
	dbg.Lvl2("DEPTH:", depth)
	dbg.Lvl2("TOTAL HOSTS:", len(hostnames))

	// generate the configuration file from the tree
	cf := config.ConfigFromTree(t, hostnames)
	cfb, err := json.Marshal(cf)
	err = ioutil.WriteFile(d.BuildDir + "/cfg.json", cfb, 0666)
	if err != nil {
		log.Fatal(err)
	}
}

/*
func main() {
	dbg.Lvl2("RUNNING DEPLOY2DETER WITH RATE:", rate, " on machines:", nmachs)

	os.MkdirAll("remote", 0777)
	readHosts()

	// killssh processes on users
	dbg.Lvl2("Stopping programs on user.deterlab.net")
	cliutils.SshRunStdout(user, host, "killall ssh scp deter 2>/dev/null 1>/dev/null")

	// If we have to build, we do it for all programs and then copy them to 'host'
	if build {
		doBuild()
	}

	killssh := exec.Command("pkill", "-f", "ssh -t -t")
	killssh.Stdout = os.Stdout
	killssh.Stderr = os.Stderr
	err := killssh.Run()
	if err != nil {
		log.Print("Stopping ssh: ", err)
	}

	calculateGraph()

	// Copy everything over to deterlabs
	err = cliutils.Rsync(user, host, "remote", "")
	if err != nil {
		log.Fatal(err)
	}

	// setup port forwarding for viewing log server
	// ssh -L 8081:pcXXX:80 username@users.isi.deterlab.net
	// ssh username@users.deterlab.net -L 8118:somenode.experiment.YourClass.isi.deterlab.net:80
	fmt.Println("setup port forwarding for master logger: ", masterLogger)
	cmd := exec.Command(
		"ssh",
		"-t",
		"-t",
		fmt.Sprintf("%s@%s", user, host),
		"-L",
		"8081:" + masterLogger + ":10000")
	err = cmd.Start()
	if err != nil {
		log.Fatal("failed to setup portforwarding for logging server")
	}

	dbg.Lvl2("runnning deter with nmsgs:", nmsgs)
	// run the deter lab boss nodes process
	// it will be responsible for forwarding the files and running the individual
	// timestamping servers
	dbg.Lvl2(cliutils.SshRunStdout(user, host,
		"GOMAXPROCS=8 remote/deter -nmsgs=" + strconv.Itoa(nmsgs) +
		" -hpn=" + strconv.Itoa(hpn) +
		" -bf=" + strconv.Itoa(bf) +
		" -rate=" + strconv.Itoa(rate) +
		" -rounds=" + strconv.Itoa(rounds) +
		" -debug=" + strconv.Itoa(debug) +
		" -failures=" + strconv.Itoa(failures) +
		" -rfail=" + strconv.Itoa(rFail) +
		" -ffail=" + strconv.Itoa(fFail) +
		" -test_connect=" + strconv.FormatBool(testConnect) +
		" -app=" + app +
		" -suite=" + suite +
		" -kill=" + strconv.FormatBool(kill)))

	dbg.Lvl2("END OF DEPLOY2DETER")
}


func doBuild() {
}
*/