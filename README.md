# `jobman` -- Run multiple jobs using multiple executors in parallel.

(This is very, very alpha software.)

The `jobman` command line tool is a simple job runner that you might
typically use if you have a small collection of desktop machines
that you want to run jobs on. Jobs and job runners are specified in
two separate YAML files.

For example, you might have two desktop machines with two GPUs each. You
can use this for running jobs using:

```YAML
runners:
  thor0: ssh thor podman run -e NVIDIA_VISIBLE_DEVICES=0 registry:5000/pytorch-img {cmd}
  thor1: ssh thor podman run -e NVIDIA_VISIBLE_DEVICES=1 registry:5000/pytorch-img {cmd}
  odin0: ssh odin podman run -e NVIDIA_VISIBLE_DEVICES=0 registry:5000/pytorch-img {cmd}
  odin1: ssh odin podman run -e NVIDIA_VISIBLE_DEVICES=1 registry:5000/pytorch-img {cmd}
```

You actually probably want the script/command on stdin:

```YAML
pre: echo running on the valhalla cluster
oninput: true
runners:
  thor0: ssh thor podman run -e NVIDIA_VISIBLE_DEVICES=0 registry:5000/pytorch-img /bin/bash
  thor1: ssh thor podman run -e NVIDIA_VISIBLE_DEVICES=1 registry:5000/pytorch-img /bin/bash
  odin0: ssh odin podman run -e NVIDIA_VISIBLE_DEVICES=0 registry:5000/pytorch-img /bin/bash
  odin1: ssh odin podman run -e NVIDIA_VISIBLE_DEVICES=1 registry:5000/pytorch-img /bin/bash
```

You specify jobs like this:

```YAML
jobs:
  - python3 train.py --seed 0
  - python3 train.py --seed 1
  - python3 train.py --seed 2
  ...
  - python3 train.py --seed 100
```

Or like this:

```YAML
pre: |
  echo startup: test jobs
  rm -rf /tmp/mylogs
logdir: /tmp/mylogs
template:
  command: echo {i}
  range: 10
```

Now you can run your jobs with:

```Bash
$ jobman -j jobs.yaml -o logs
```

Jobman will execute your jobs on all the different runners and place the log
files in the logs subdirectory.

You can also just specify jobs from the command line. This will run 100 jobs
with seed 0..99:

```Bash
$ jobman -T 'python3 train.py --seed={i}' -R 100 -o logs
```


Usage:

```
Usage:
  jobman [OPTIONS]

Application Options:
  -v, --verbose       Verbose output
  -w, --wait=         Wait after each job completion.
  -r, --runners=      Runners file (default: env JOBMAN_RUNNERS or runners.yaml)
  -l, --line-buffer=  Line buffer size. (default: 1)
  -t, --line-timeout= Line timeout. (default: 1)
  -o, --log-dir=      Log directory.
  -j, --jobs=         Jobs file
  -T, --template=     Command template
  -R, --range=        Range used with command template.

Help Options:
  -h, --help          Show this help message
```

# TODO

- override some command line options from jobs file
- add `pre:` and `post:` section to both runners and jobs
- add error handling options and retry options
- allow non-shell, list-based invocation of commands
- allow templating and multiline scripts inside jobs
- command on stdin
- jsonrun
- job specs -- input, output, commandline, time limit, resource limit
