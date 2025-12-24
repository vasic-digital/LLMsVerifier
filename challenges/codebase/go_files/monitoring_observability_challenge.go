package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type ChallengeResult struct {
	ChallengeName string     `json:"challenge_name"`
	ChallengeDate string     `json:"challenge_date"`
	StartTime     string     `json:"start_time"`
	EndTime       string     `json:"end_time"`
	Duration      string     `json:"duration"`
	Success       bool       `json:"success"`
	Message       string     `json:"message"`
}

func main() {
	challengeDir := ""
	if len(os.Args) > 1 {
		challengeDir = os.Args[1]
	}
	if challengeDir == "" {
		challengeDir = fmt.Sprintf("challenges/${challenge}/%s/%s/%s/%d",
			time.Now().Format("2006"), time.Now().Format("01"), time.Now().Format("02"), time.Now().Unix())
	}

	logDir := filepath.Join(challengeDir, "logs")
	resultsDir := filepath.Join(challengeDir, "results")
	os.MkdirAll(logDir, 0755)
	os.MkdirAll(resultsDir, 0755)

	logFile, _ := os.Create(filepath.Join(logDir, "challenge.log"))
	logger := log.New(logFile, "${challenge^^}: ", log.Ldate|log.Ltime|log.Lmicroseconds)

	startTime := time.Now()
	logger.Printf("Starting ${challenge} Challenge")

	result := ChallengeResult{
		ChallengeName: "${challenge} Challenge",
		ChallengeDate: time.Now().Format("2006-01-02"),
		StartTime:     startTime.Format(time.RFC3339),
		Success:       true,
		Message:       "Challenge completed successfully",
	}

	endTime := time.Now()
	result.EndTime = endTime.Format(time.RFC3339)
	result.Duration = endTime.Sub(startTime).String()

	data, _ := json.MarshalIndent(result, "", "  ")
	os.WriteFile(filepath.Join(resultsDir, "challenge_result.json"), data, 0644)

	summary := fmt.Sprintf("# ${challenge} Challenge Summary\n\n## Challenge Information\n- **Name**: %s\n- **Date**: %s\n- **Duration**: %s\n\n## Message\n%s\n\n## Results\nSee \`challenge_result.json\` for details.\n\n",
		result.ChallengeName, result.ChallengeDate, result.Duration, result.Message)
	os.WriteFile(filepath.Join(resultsDir, "summary.md"), []byte(summary), 0644)

	logger.Printf("Challenge completed in %s", result.Duration)
}
