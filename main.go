package main

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

type TaskStatus string

const (
	Done       TaskStatus = "done"
	Todo       TaskStatus = "todo"
	InProgress TaskStatus = "in-progress"
)

type Task struct {
	DeletedAt time.Time `json:"deletedAt"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	Id          int        `json:"id"`
}

var filePath string = "./db/tasks.json"

func main() {
	dbInit()

	command := os.Args[1]

	if command == "" {
		return
	}

	switch command {
	case "add":
		if len(os.Args) < 3 {
			panic("missing arguments")
		}
		add(os.Args[2])

	case "update":
		if len(os.Args) < 4 {
			panic("missing arguments")
		}

		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			panic(err)
		}

		update(id, os.Args[3])
	case "delete":
		if len(os.Args) < 3 {
			panic("missing arguments")
		}

		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			panic(err)
		}

		delete(id)
	case "mark-in-progress":
		if len(os.Args) < 3 {
			panic("missing arguments")
		}

		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			panic(err)
		}
		updateTaskStatus(id, InProgress)

	case "mark-done":
		if len(os.Args) < 3 {
			panic("missing arguments")
		}

		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			panic(err)
		}
		updateTaskStatus(id, Done)

	case "list":
		var status TaskStatus
		if len(os.Args) > 2 {
			status = TaskStatus(os.Args[2])
		}
		list(status)

	case "help":
	case "-help":
	case "--help":
	default:
		// show help
	}
}

func dbInit() {
	if _, err := os.Stat(filePath); !errors.Is(err, os.ErrNotExist) {
		return
	}

	err := os.MkdirAll(strings.Replace(filePath, "/tasks.json", "", 1), 0750)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(filePath, []byte("[]"), 0660)
	if err != nil {
		panic(err)
	}
}

func getTasks() []Task {
	data, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	var taskList []Task
	json.Unmarshal(data, &taskList)

	n := 0
	for _, val := range taskList {
		if time.Time.IsZero(val.DeletedAt) {
			taskList[n] = val
			n++
		}
	}

	taskList = taskList[:n]

	return taskList
}

func updateTasks(tasks []Task) {
	data, err := json.Marshal(tasks)
	if err != nil {
		panic(err)
	}

	if err = os.WriteFile(filePath, data, 0660); err != nil {
		panic(err)
	}
}

func getTask(tasks []Task, taskId int) (*Task, error) {
	n, found := slices.BinarySearchFunc(tasks, Task{
		Id: taskId,
	}, func(a, b Task) int {
		return cmp.Compare(a.Id, b.Id)
	})

	if !found {
		return &Task{}, errors.New("Task was not found")
	}

	return &tasks[n], nil
}

func add(title string) {
	taskId := 1

	tasks := getTasks()
	if len(tasks) > 0 {
		taskId = tasks[len(tasks)-1].Id + 1
	}

	tasks = append(tasks, Task{
		Id:          taskId,
		Description: title,
		Status:      Todo,
		CreatedAt:   time.Now(),
	})

	updateTasks(tasks)
	list("")
}

func update(taskId int, description string) {
	tasks := getTasks()
	task, err := getTask(tasks, taskId)
	if err != nil {
		panic(err)
	}

	if !task.DeletedAt.IsZero() {
		fmt.Println("Unable to update deleted tasks!")
		return
	}

	if task.Status == Done {
		fmt.Println("Unable to update completed tasks!")
		return
	}

	if task.Description != description {
		task.Description = description
		task.UpdatedAt = time.Now()
		updateTasks(tasks)
	}

	list("")
}

func updateTaskStatus(taskId int, status TaskStatus) {
	tasks := getTasks()
	task, err := getTask(tasks, taskId)
	if err != nil {
		panic(err)
	}

	if status == Done && task.Status != InProgress {
		fmt.Println("Task should be in-progress before done!")
		return
	}

	if status == InProgress && task.Status != Todo {
		fmt.Println("Task should be todo before in-progress!")
		return
	}

	if !task.DeletedAt.IsZero() {
		fmt.Println("Unable to update deleted tasks!")
		return
	}

	if task.Status != status {
		task.Status = status
		task.UpdatedAt = time.Now()
		updateTasks(tasks)
	}

	list("")
}

func delete(taskId int) {
	tasks := getTasks()
	task, err := getTask(tasks, taskId)
	if err != nil {
		return
	}

	if !task.DeletedAt.IsZero() {
		return
	}

	if task.Status == Done {
		fmt.Println("Unable to update completed tasks!")
		return
	}

	task.DeletedAt = time.Now()
	updateTasks(tasks)
	list("")
}

func printTable(tasks []Task) {
	tw := tabwriter.NewWriter(os.Stdout, 1, 1, 2, ' ', 0)
	fmt.Fprint(tw, "Id\tDescription\tStatus\tCreated at\tUpdated at\n")

	for _, task := range tasks {
		var updatedAt string
		if task.UpdatedAt.IsZero() {
			updatedAt = "-"
		} else {
			updatedAt = task.UpdatedAt.Format(time.RFC822)
		}

		fmt.Fprintf(
			tw,
			"%v\t%v\t%v\t%v\t%v\n",
			task.Id,
			task.Description,
			task.Status,
			task.CreatedAt.Format(time.RFC822),
			updatedAt,
		)
	}

	tw.Flush()
}

func list(status ...TaskStatus) {
	statusToGet := status[0]
	tasks := getTasks()
	if statusToGet == "" {

		n := 0
		for _, val := range tasks {
			if val.Status != Done {
				tasks[n] = val
				n++
			}
		}

		tasks = tasks[:n]
		printTable(tasks)
		return
	}

	n := 0
	for _, val := range tasks {
		if val.Status == statusToGet {
			tasks[n] = val
			n++
		}
	}

	tasks = tasks[:n]
	printTable(tasks)
}
