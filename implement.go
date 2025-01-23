package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

func LineContainsPrefix(line, prefix string) bool {
	return len(line) >= len(prefix) && prefix == line[:len(prefix)]
}

func main() {
	trainerPath := "C:/Users/Administrator/Online3D/3DGS/gaussian-splatting/train_video.py"

	cmd := exec.Command("python3", trainerPath)
	cmd.Env = append(os.Environ(), "PYTHONPATH=C:/Users/Administrator/Online3D/3DGS/gaussian-splatting/envs/gaussian_splatting")
	err := os.Chdir(filepath.Dir(trainerPath))
	if err != nil {
		fmt.Println("Error changing directory:", err)
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Error creating stdout pipe:", err)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting command:", err)
		return
	}

	fmt.Println("Training started. Waiting for completion...")

	var indexPath string
	var wg sync.WaitGroup
	wg.Add(1)
	done := make(chan bool)
	go func() {
		defer wg.Done()
		prefix := "Output folder:"
		stdOutScanner := bufio.NewScanner(stdout)
		for stdOutScanner.Scan() {
			if LineContainsPrefix(stdOutScanner.Text(), prefix) {
				line := stdOutScanner.Text()
				first := strings.Index(line, "./output/")
				if first == -1 {
					fmt.Println("Output folder find error")
					return
				}
				indexPath = line[first : first+19]
			}
			fmt.Println(stdOutScanner.Text())
		}
		done <- true
	}()

	go func() {
		wg.Wait()
		if err := cmd.Wait(); err != nil {
			fmt.Println("Command finished with error:", err)
			done <- false
		} else {
			fmt.Println("Training completed successfully")
		}
	}()

	success := <-done
	if success {
		fmt.Println("Training completed successfully")
	} else {
		fmt.Println("Training finished without expected output")
	}

	indexPathAbs, err := filepath.Abs(indexPath)
	if err != nil {
		fmt.Println("Error to convert path:", err)
		return
	}

	viewerPath := "C:/Users/Administrator/Online3D/3DGS/gaussian-splatting/SIBR_viewer.py"
	err = os.Chdir(filepath.Dir(viewerPath))

	cmd = exec.Command("python3", filepath.Base(viewerPath), indexPathAbs)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error:", err)
		fmt.Println("Output:", string(output))
		return
	}
	fmt.Println("viewer loading complete")
}
