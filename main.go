package main

import (
	"io"
	"errors"
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
	Wait int `short:"w" long:"wait" description:"Wait after each job completion."`
	Runners string `short:"r" long:"runners" description:"Runners file (default: env JOBMAN_RUNNERS or runners.yaml)" default:""`
	LineBuffer int `short:"l" long:"line-buffer" description:"Line buffer size." default:"1"`
	LineTimeout float32 `short:"t" long:"line-timeout" description:"Line timeout." default:"1"`
	LogDir string `short:"o" long:"log-dir" description:"Log directory." default:""`
	Jobs string `short:"j" long:"jobs" description:"Jobs file" default:""`
	Template string `short:"T" long:"template" description:"Command template" default:""`
	Range string `short:"R" long:"range" description:"Range used with command template." default:""`
}

var Parser = flags.NewParser(&options, flags.Default)

var TIMEOUT = errors.New("timeout")

func ReadLineWithTimeout(r *bufio.Reader, timeout time.Duration) (string, error) {
	line, err := r.ReadString('\n')
	return line, err
	// FIXME
	// t := time.NewTimer(timeout)
	// defer t.Stop()
	// for {
	// 	select {
	// 	case <-t.C:
	// 		fmt.Println("timeout")
	// 		return "", TIMEOUT
	// 	default:
	// 		line, err := r.ReadString('\n')
	// 		if err == io.EOF {
	// 			return line, io.EOF
	// 		}
	// 		if err != nil {
	// 			return "", err
	// 		}
	// 		return line, nil
	// 	}
	// }
}

func LinewiseOutput(prefix string, eofnotify bool) *io.PipeWriter {
	prefix = "[" + prefix + "] "
	reader, writer := io.Pipe()
	buffered_reader := bufio.NewReader(reader)
	go func() {
		lines := []string{}
		last := time.Now()
		for {
			line, err := ReadLineWithTimeout(buffered_reader, 1 * time.Second)
			if err != TIMEOUT {
				if err != nil {
					break
				}
				lines = append(lines, line)
				last = time.Now()
			}
			// fmt.Println("<", time.Since(last), ">")
			if len(lines) >= options.LineBuffer || line == "\n" || time.Since(last) > time.Second {
				fmt.Print(prefix, strings.Join(lines, prefix))
				lines = []string{}
			}
		}
		if len(lines) > 0 {
			fmt.Print(prefix, strings.Join(lines, prefix))
		}
		if eofnotify {
			fmt.Println(prefix, "---")
		}
	} ()
	return writer
}

type jobdesc struct {
	name string
	command string
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
		job := item.(jobdesc)
		actual := strings.Replace(cmd, "{cmd}", job.command, -1)
		ident := fmt.Sprintf("%s@%s", job.name, name)
		fmt.Printf("# starting: %s :: %s\n", ident, actual)
		cmd := exec.Command("/bin/sh", "-c", actual)
		if options.LogDir != "" {
			logfile := fmt.Sprintf("%s/%s_%010d.log", options.LogDir, ident, time.Now().Unix())
			stream, err := os.Create(logfile)
			if err != nil {
				fmt.Printf("# failed to create log file: %s\n", logfile)
			}
			defer stream.Close()
			cmd.Stdout = stream 
			cmd.Stderr = stream
		} else {
			stream := LinewiseOutput(ident, true)
			defer stream.Close()
			cmd.Stdout = stream 
			cmd.Stderr = stream
		}
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

	if options.LogDir != "" {
		err := os.MkdirAll(options.LogDir, 0755)
		if err != nil {
			fmt.Printf("# failed to create log directory: %s\n", options.LogDir)
		}
	}

	if options.Runners == "" {
		options.Runners = os.Getenv("JOBMAN_RUNNERS")
		if options.Runners == "" {
			options.Runners = "runners.yaml"
		}
	}

	yrunners, err := ReadYaml(options.Runners)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	queue := fifo.NewQueue()

	if options.Jobs != "" {
		yjobs, err := ReadYaml(options.Jobs)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		switch jobs := yjobs["jobs"].(type) {
		case []interface{}:
			for k, v := range jobs {
				job := jobdesc{name: strconv.Itoa(k), command: v.(string)}
				queue.Add(job)
			}
		case map[string]interface{}:
			for k, v := range jobs {
				job := jobdesc{name: k, command: v.(string)}
				queue.Add(job)
			}
		default:
			fmt.Println("jobs is not a list or map")
			os.Exit(1)
		}
	}


	if options.Template != "" {
		if options.Range == "" {
			job := jobdesc{name: "job", command: options.Template}
			queue.Add(job)
		} else if n, err := strconv.Atoi(options.Range); err == nil {
			for i := 0; i < n; i++ {
				cmdline := strings.Replace(options.Template, "{i}", strconv.Itoa(i), -1)
				job := jobdesc{name: strconv.Itoa(i), command: cmdline}
				queue.Add(job)
			}
		} else {
			fmt.Println("range is not a number")
			os.Exit(1)
		}
	}


	if queue.Len() == 0 {
		fmt.Println("no jobs to run")
		os.Exit(1)
	}


	wg := sync.WaitGroup{}
	switch runners := yrunners["runners"].(type) {
	case []interface{}:
		for k, v := range runners {
			wg.Add(1)
			go func(k int, v string) {
				Runner(strconv.Itoa(k), v, queue)
				wg.Done()
			} (k, v.(string))
		}
	case map[string]interface{}:
		for k, v := range runners {
			wg.Add(1)
			go func(k string, v string) {
				Runner(k, v, queue)
				wg.Done()
			} (k, v.(string))
		}
	default:
		fmt.Println("runners is not a list or map")
		os.Exit(1)
	}
	wg.Wait()
	fmt.Println("done")
}
