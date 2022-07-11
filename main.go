package main

import (
	"io"
	"bufio"
	"os"
	"fmt"
	"time"
	"os/exec"
	"strconv"
	"github.com/jessevdk/go-flags"
	"io/ioutil"
	"strings"
	"sync"
	"github.com/foize/go.fifo"
	"gopkg.in/yaml.v3"
)

var options struct {
	Verbose bool `short:"v" long:"verbose" description:"Verbose output"`
	Wait int `short:"w" long:"wait" description:"Wait time"`
	Runners string `short:"r" long:"runners" description:"Runners file (default: env JOBMAN_RUNNERS or runners.yaml)" default:""`
	Args struct {
		Jobs string `description:"Jobs file"`
	} `positional-args:"yes" required:"yes"`
}

var Parser = flags.NewParser(&options, flags.Default)

func LinewiseOutput(prefix string) io.Writer {
	reader, writer := io.Pipe()
	buffered_reader := bufio.NewReader(reader)
	go func() {
		for {
			line, err := buffered_reader.ReadString('\n')
			if err != nil {
				break
			}
			fmt.Printf("[%s] %s", prefix, line)
		}
	} ()
	return writer
}

func Runner(name string, cmd string, queue *fifo.Queue) {
	if options.Verbose {
		fmt.Printf("runner: %s :: %s\n", name, cmd)
	}
	for {
		item := queue.Next()
		if item == nil {
			break
		}
		actual := strings.Replace(cmd, "{cmd}", item.(string), -1)
		fmt.Println("[", name, "]", actual)
		cmd := exec.Command("/bin/sh", "-c", actual)
		cmd.Stdout = LinewiseOutput(name)
		cmd.Stderr = LinewiseOutput(name+"_err")
		cmd.Run()
		time.Sleep(1 * time.Second)
	}
}

func ReadYaml(file string) (map[interface{}]interface{}, error) {
	text, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	data := make(map[interface{}]interface{})
	err = yaml.Unmarshal(text, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func main() {
	if _, err := Parser.Parse(); err != nil {
		os.Exit(1)
	}

	if options.Runners == "" {
		options.Runners = os.Getenv("JOBMAN_RUNNERS")
		if options.Runners == "" {
			options.Runners = "runners.yaml"
		}
	}

	yjobs, err := ReadYaml(options.Args.Jobs)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	yrunners, err := ReadYaml(options.Runners)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	queue := fifo.NewQueue()

	jobs := yjobs["jobs"].([]interface{})
	for k, v := range jobs {
		if options.Verbose {
			fmt.Printf("adding: %d: %s\n", k, v.(string))
		}
		queue.Add(v)
	}

	runners := yrunners["runners"].([]interface{})
	wg := sync.WaitGroup{}
	for k, v := range runners {
		if options.Verbose {
			fmt.Printf("adding: %d: %s\n", k, v.(string))
		}
		wg.Add(1)
		go func(k int, v string) {
			Runner(strconv.Itoa(k), v, queue)
			wg.Done()
		} (k, v.(string))
	}
	wg.Wait()
	fmt.Println("done")
}
