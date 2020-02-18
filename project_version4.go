package main

import (
	"fmt"
	_ "net/http/pprof"
	"sync"

	"github.com/pkg/profile"
)

//Params struct
type Params struct {
	glb         sync.WaitGroup
	jobs        chan Job
	rootName    string
	noOfWorkers int
	wt          Weighted
}

const KB = 1024
const MB = 1024 * KB
const GB = 1024 * MB
const TB = 1024 * GB

func main() {
	defer profile.Start(profile.CPUProfile).Stop()
	//defer profile.Start(profile.MemProfile).Stop()
	var c conf
	c.GetConf()

	for x := 0; x < len(c.Path); x++ {
		var p Params
		gtree := Newtree(c.Path[x])
		p.rootName = c.Path[x]
		p.noOfWorkers = 2

		//setting the counter
		p.NewWeighted()
		var j = make(chan Job, 10000)
		p.jobs = j
		//pushing root node to the channel
		p.jobs <- Job{gtree.Root, nil}

		fmt.Println("Creation of tree starts....")
		p.CreateWorkerPool()
		fmt.Println("Tree created successfully....")
		//gtree.Root.printTree()
		gtree.PrintTreeInJSON(gtree.Root.Name, int64(2*GB), c.TreeName[x], c.BackupfileName[x])
		files, folders, size := gtree.Root.Info.FileCount, gtree.Root.Info.FolderCount, gtree.Root.Info.Size

		bkpsets := f1(gtree.Root, int64(9))

		//Printing the created backupsets
		for i := 0; i < len(bkpsets); i++ {
			fmt.Println("--------------Backupset: ", i, "---------------------")
			fmt.Println("FileCount: ", bkpsets[i].Count)
			fmt.Println("Paths: ")
			for j := 0; j < len(bkpsets[i].Paths); j++ {
				fmt.Println(bkpsets[i].Paths[j])

			}
		}

		if files == int64(c.Files[x]) {
			fmt.Printf("Success,expected %d,got %d", c.Files[x], files)
			fmt.Println()
		} else {
			fmt.Printf("failed,expected %d,got %d", c.Files[x], files)
			fmt.Println()
		}

		if folders == int64(c.Folders[x]) {
			fmt.Printf("Success,expected %d,got %d", c.Folders[x], folders)
			fmt.Println()
		} else {
			fmt.Printf("failed,expected %d,got %d", c.Folders[x], folders)
			fmt.Println()
		}

		if size == int64(c.Size[x]) {
			fmt.Printf("Success,expected %d,got %d", c.Size[x], size)
			fmt.Println()
		} else {
			fmt.Printf("failed,expected %d,got %d", c.Size[x], size)
			fmt.Println()
		}
		/*
			//creating backupsets

			bkpsets := BackupCreation2(gtree.Root, int64(c.Backupsize[x]))
			//fmt.Println(bkpsets)

			//Printing the created backupsets
			for i := 0; i < len(bkpsets); i++ {
				fmt.Println("--------------Backupset: ", i, "---------------------")
				fmt.Println("size: ", bkpsets[i].Size)
				fmt.Println("Paths: ")
				for j := 0; j < len(bkpsets[i].Paths); j++ {
					fmt.Println(bkpsets[i].Paths[j])

				}
			}*/

	}
}
