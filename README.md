# Cube

**Cube** is a container orchestration tool, designed to run tasks (containers) across multiple nodes, and can be controlled via a command-line interface (CLI). This project was inspired by the book *[Building an Orchestrator with Go](https://learning.oreilly.com/library/view/build-an-orchestrator/9781617299759/)*, with several custom modifications and enhancements.

## Features

- **Multi-node orchestration**: Run containers across multiple worker nodes.
- **Custom scheduling**: Choose the type of scheduler that best suits your needs.
- **CLI-controlled**: Easily manage your orchestrator with intuitive command-line options.

## Build Instructions

To build the project, simply run:

```bash
go build
```

This will compile the Cube binary.

## Worker Node Setup

When you run `./cube`, you'll see a list of available commands. To begin, you'll need to start a worker node.

Run:

```bash
cube worker --help
```

This will give you a list of options to configure the worker. You can also start a default worker (running on port `5555` and IP `0.0.0.0`) by simply running:

```bash
cube worker
```

## Manager Node Setup

Once your worker is running, you need to start a manager to register and manage the workers. Run:

```bash
cube manager --help
```

This command will show you all the available options for setting up the manager. If you’ve run the worker with default settings, you can start a manager without specifying additional options:

```bash
cube manager
```

Alternatively, if you have custom worker configurations, you'll need to specify the list of workers, scheduler type, storage type, and the host/port for the manager.

## Running a Task

After the manager is set up, you can run a task. First, check the available options with:

```bash
cube run --help
```

You'll see that you need to provide a task template and the manager's URL. If you've used the default manager, the manager URL is not a concern. By default, Cube will use the `task.json` file for the task template.

An example of a task template looks like this:

```json
{
  "ID": "266592cd-960d-4091-981c-8c25c44b1012",
  "State": 2,
  "Task": {
      "State": 1,
      "ID": "266592cd-960d-4091-981c-8c25c44b1012",
      "Name": "echo",
      "Image": "cplk01/echo-server",
      "ExposedPorts": {
          "8080/tcp": {}
      },
      "PortBindings": {
          "8080/tcp": "8080"
      },
      "HealthCheck": "/health"
  }
}
```

## Task Status

After scheduling the task, you’ll want to check its status. You can do this by running:

```bash
cube status
```

You can also specify which manager to check for the status of the tasks.

## Node Information

You can check the nodes (workers) in your cluster, their running tasks, and their load by running:

```bash
cube nodes
```

This command works similarly to `cube status`, allowing you to specify a manager for the node status information.