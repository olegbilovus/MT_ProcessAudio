package questdb

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func InitQuestDB(URL string, experimentName string) error {
	if res, err := http.Get(URL + "/exec?query=" + url.QueryEscape("SELECT NOW();")); err != nil || res.StatusCode > 299 {
		return fmt.Errorf("unable to ping database. status code: %d, err :%v", res.StatusCode, err)
	}

	const queryCreateAudio = `
		CREATE TABLE IF NOT EXISTS %s (
			ts TIMESTAMP,
			audio_level DOUBLE,
			name SYMBOL   
		) TIMESTAMP(ts) PARTITION BY DAY WAL;`

	if err := DeleteTable(URL, GetAudioTableName(experimentName)); err != nil {
		return err
	}
	if err := CreateTable(URL, GetAudioTableName(experimentName), queryCreateAudio); err != nil {
		return err
	}

	const queryCreateTranscript = `
		CREATE TABLE IF NOT EXISTS %s (
			ts_start TIMESTAMP,
			duration DOUBLE,
			ts_end TIMESTAMP,
			word VARCHAR,
			name SYMBOL
		) TIMESTAMP(ts_start) PARTITION BY DAY WAL;`

	if err := DeleteTable(URL, GetTranscriptTableName(experimentName)); err != nil {
		return err
	}
	if err := CreateTable(URL, GetTranscriptTableName(experimentName), queryCreateTranscript); err != nil {
		return err
	}

	return nil
}

func GetAudioTableName(experimentName string) string {
	return "audio_" + experimentName
}

func CreateTable(URL string, tableName string, query string) error {
	queryComplete := fmt.Sprintf(query, tableName)

	resp, err := http.Get(URL + "/exec?query=" + url.QueryEscape(queryComplete))
	if resp.StatusCode > 299 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error creating table %s, status: %s, body: %s", tableName, resp.Status, string(body))
	}
	return err
}

func DeleteTable(URL string, tableName string) error {
	const query = `DROP TABLE IF EXISTS %s;`

	queryComplete := fmt.Sprintf(query, tableName)

	resp, err := http.Get(URL + "/exec?query=" + url.QueryEscape(queryComplete))
	if resp.StatusCode > 299 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error dropping table %s, status: %s, body: %s", tableName, resp.Status, string(body))
	}

	return err
}

func GetTranscriptTableName(experimentName string) string {
	return "transcript_" + experimentName
}
