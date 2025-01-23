package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

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

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println("Error creating stderr pipe:", err)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting command:", err)
		return
	}

	fmt.Println("Training started. Waiting for completion...")

	var wg sync.WaitGroup
	wg.Add(2)
	done := make(chan bool)
	go func() {
		defer wg.Done()
		stdOutScanner := bufio.NewScanner(stdout)
		for stdOutScanner.Scan() {
			fmt.Println(stdOutScanner.Text())
		}
	}()

	go func() {
		defer wg.Done()
		stdErrScanner := bufio.NewScanner(stderr)
		for stdErrScanner.Scan() {
			fmt.Println(stdErrScanner.Text())
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

	viewerPath := "C:/Users/Administrator/Online3D/3DGS/gaussian-splatting/SIBR_viewer.py"

	err = os.Chdir(filepath.Dir(viewerPath))
	if err != nil {
		fmt.Println("Error changing directory:", err)
		return
	}

	cmd = exec.Command("python3", filepath.Base(viewerPath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error:", err)
		fmt.Println("Output:", string(output))
		return
	}
	fmt.Println("viewer loading complete")
}
