package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/questdb/go-questdb-client/v4"
	log "github.com/sirupsen/logrus"

	questdbinit "github.com/olegbilovus/MT_ProcessAudio/internal/questdb"
)

type LogEvent struct {
	Time               time.Time `csv:"time"`
	Name               string    `csv:"name"`
	AudioDataFile      string    `csv:"audio_data_file"`
	SampleRate         int       `csv:"audio_sample_rate"`
	TranscriptDataFile string    `csv:"transcript_data_file"`
}

//goland:noinspection D
func main() {
	log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true,
		PadLevelText: true,
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			filename := path.Base(frame.File) + ":" + strconv.Itoa(frame.Line)
			return "", filename
		}})
	log.SetLevel(log.InfoLevel)

	var name = flag.String("name", "", "name of the experiment. It will overwrite any existing ones")
	var logEventFilePath = flag.String("log-event-file", "", "path to the csv log events data")
	var audioDataFileDir = flag.String("audio-data-file-dir", "", "path to the csv audio data file dir containing the files exported from Sonic Visualiser")
	var transcriptFileDir = flag.String("transcript-file-dir", "", "path to the csv audio transcript file. Expected headers: TIME,VALUE,DURATION,LABEL. VALUE will be ignored")
	var skipAudioData = flag.Bool("skip-audio-data", false, "skip the process and upload of audio data file")

	flag.Parse()

	if *skipAudioData {
		log.Warning("Skipping audio data file process and upload")
	}

	var err error

	experimentName := *name
	if len(experimentName) == 0 {
		experimentName = filepath.Base(*logEventFilePath)
		experimentName = strings.TrimSuffix(experimentName, filepath.Ext(experimentName))
	}

	if err := questdbinit.InitQuestDB("http://127.0.0.1:9000", experimentName); err != nil {
		log.Fatalf("failed to init QuestDB: %v", err)
	}

	ctx := context.TODO()

	client, err := questdb.LineSenderFromConf(ctx, "http::addr=localhost:9000;username=admin;password=quest;retry_timeout=0")
	if err != nil {
		log.Fatalf("failed to create QuestDB client: %v", err)
	}
	defer client.Close(ctx)

	logEventData, err := getDataFromCSV[LogEvent](*logEventFilePath)
	if err != nil {
		log.Fatalf("error parsing log event data: %v", err)
	}

	audioDataTableName := questdbinit.GetAudioTableName(experimentName)
	transcriptDataTableName := questdbinit.GetTranscriptTableName(experimentName)
	countAudio, countTranscript := 0, 0
	for _, logEvent := range logEventData {
		var audioDataArray []*AudioData
		if !*skipAudioData {
			audioDataArray, err = processAudioData(path.Join(*audioDataFileDir, logEvent.AudioDataFile), logEvent.Time, logEvent.SampleRate, logEvent.Name)
			if err != nil {
				log.Fatalf("error processing audio data %s: %v", logEvent.AudioDataFile, err)
			}

			for _, audioData := range audioDataArray {
				err := client.Table(audioDataTableName).
					Symbol("name", audioData.Name).
					Float64Column("audio_level", audioData.AudioLevel).
					At(ctx, audioData.Time)
				if err != nil {
					log.Fatalln(err)
				}
			}
		}

		transcriptDataArray, err := processTranscriptData(path.Join(*transcriptFileDir, logEvent.TranscriptDataFile), logEvent.Time, logEvent.Name)
		if err != nil {
			log.Fatalf("error processing transcript data %s: %v", logEvent.TranscriptDataFile, err)
		}

		for _, transcriptData := range transcriptDataArray {
			err := client.Table(transcriptDataTableName).
				Symbol("name", transcriptData.Name).
				Float64Column("duration", transcriptData.Duration).
				TimestampColumn("ts_end", transcriptData.EndTime).
				StringColumn("word", transcriptData.Word).
				At(ctx, transcriptData.StartTime)
			if err != nil {
				log.Fatalln(err)
			}
		}

		if !*skipAudioData {
			log.WithField("name", logEvent.Name).Infof("Saved audio data: %d, transcript data: %d", len(audioDataArray), len(transcriptDataArray))

			countAudio += len(audioDataArray)
		} else {
			log.WithField("name", logEvent.Name).Infof("Skipped audio data, saved transcript data: %d", len(transcriptDataArray))
		}

		countTranscript += len(transcriptDataArray)
	}
	client.Flush(ctx)

}

func getDataFromCSV[T any](csvFile string) ([]*T, error) {
	file, err := os.OpenFile(csvFile, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %v", csvFile, err)
	}
	defer file.Close()

	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("error going to the start of the file %s: %v", csvFile, err)
	}

	data := make([]*T, 0)
	if err := gocsv.UnmarshalFile(file, &data); err != nil {
		return nil, fmt.Errorf("error parsing csv file %s: %v", csvFile, err)
	}

	return data, nil
}

type TranscriptData struct {
	StartSeconds float64 `csv:"TIME"`
	Duration     float64 `csv:"DURATION"`
	Word         string  `csv:"LABEL"`
	StartTime    time.Time
	EndTime      time.Time
	Name         string
}

func processTranscriptData(csvFile string, startData time.Time, name string) ([]*TranscriptData, error) {
	data, err := getDataFromCSV[TranscriptData](csvFile)
	if err != nil {
		return nil, err
	}

	for _, transcriptData := range data {
		transcriptData.StartTime = startData.Add(time.Duration(transcriptData.StartSeconds*1_000_000) * time.Microsecond)
		transcriptData.EndTime = transcriptData.StartTime.Add(time.Duration(transcriptData.Duration*1_000_000) * time.Microsecond)
		transcriptData.Name = name
	}

	return data, nil

}

type AudioData struct {
	Frame      int       `csv:"frame"`
	AudioLevel float64   `csv:"audio_level"`
	Time       time.Time `csv:"ts"`
	Name       string
}

func processAudioData(csvFile string, startData time.Time, sampleRate int, name string) ([]*AudioData, error) {
	data, err := getDataFromCSV[AudioData](csvFile)
	if err != nil {
		return nil, err
	}

	intervalMicroseconds := (1 / float64(sampleRate)) * 1_000_000

	for _, audioData := range data {
		audioData.Time = startData.Add(time.Duration(float64(audioData.Frame)*intervalMicroseconds) * time.Microsecond)
		audioData.Name = name
	}

	return data, nil
}
