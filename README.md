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

You specify jobs like this:

```YAML
jobs:
  - python3 train.py --seed 0
  - python3 train.py --seed 1
  - python3 train.py --seed 2
  ...
  - python3 train.py --seed 100
```

Now you can run your jobs with:

```Bash
$ jobman jobs.yaml -o logs
```

Jobman will execute your jobs on all the different runners and place the log
files in the logs subdirectory.

Usage:

```
Usage:
  jobman [OPTIONS] Jobs

Application Options:
  -v, --verbose       Verbose output
  -w, --wait=         Wait time after each job.
  -r, --runners=      Runners file (default: env JOBMAN_RUNNERS or runners.yaml)
  -l, --line-buffer=  Line buffer size. (default: 1)
  -t, --line-timeout= Line timeout. (default: 1)
  -o, --log-dir=      Log directory.

Help Options:
  -h, --help          Show this help message

Arguments:
  Jobs:               Jobs file
```
