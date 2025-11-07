# MT_ProcessAudio

**MT_ProcessAudio** is a Go application designed to automate the ingestion, processing, and
analytics-ready import of audio signal data and their corresponding transcripts
into [QuestDB](https://questdb.io/), a high-performance time series database.

- **Audio & Transcript Parsing**:
    - Reads audio event logs, per-frame audio data, and transcript CSV files, as produced by tools
      such as Sonic Visualiser or other audio processing pipelines.
- **Time Alignment**:
    - Adjusts all data to a shared experiment timeline so that both audio levels and transcript
      words can be queried by time.
- **QuestDB Integration**:
    - Creates/overwrites two tables in QuestDB: one for audio frame data and one for transcript
      segments/labels.
- **Batch Import**:
    - Efficiently processes and imports large batches of CSVs in a single, reproducible run.
- **Flexible Configuration**:
    - Command-line flags to specify experiment naming, input locations, and to skip audio or
      transcript processing if desired.

---

## Usage

### Prerequisites

- Go 1.25+
- Running instance of [QuestDB](https://questdb.io/) at `http://127.0.0.1:9000`
- CSV files containing:
    - A log event file listing the audio/trancript data files, aligned with experiment start times
      and sample rates
    - Per-frame audio level data files
    - Transcript CSVs with headers: `TIME,VALUE,DURATION,LABEL` (VALUE can be ignored by the app)

---

### Quick Start

1. **Build the application:**

    ```bash
    git clone https://github.com/olegbilovus/MT_ProcessAudio.git
    cd MT_ProcessAudio
    go build -o mt_processaudio ./cmd/MT_ProcessAudio
    ```
2. **Prepare your CSVs and directory structure.**

3. **Run:**

    ```bash
    ./mt_processaudio \
      -log-event-file path/to/log_events.csv \
      -audio-data-file-dir path/to/audio_data/ \
      -transcript-file-dir path/to/transcripts/ \
      -name my_experiment
    ```

   **Flags:**

    - `-log-event-file`: (required) Path to the main log events CSV
    - `-audio-data-file-dir`: Directory containing audio frame CSVs (from tools like Sonic
      Visualiser)
    - `-transcript-file-dir`: Directory containing transcript CSVs
    - `-name`: Name of experiment/table prefix in QuestDB (default: log event file's basename)
    - `-skip-audio-data`: If set, skips uploading audio signals and only uploads transcripts

---

## QuestDB Table Schemas

### Audio:

| Column      | Type      | Description                       |
|-------------|-----------|-----------------------------------|
| ts          | TIMESTAMP | Time of the audio sample          |
| audio_level | DOUBLE    | Signal level (amplitude/db/other) |
| name        | SYMBOL    | Label/name for the run            |

### Transcript:

| Column   | Type      | Description                           |
|----------|-----------|---------------------------------------|
| ts_start | TIMESTAMP | Start time of transcripted word/label |
| duration | DOUBLE    | Duration of the label/word            |
| ts_end   | TIMESTAMP | End time of label                     |
| word     | VARCHAR   | Transcripted word or label            |
| name     | SYMBOL    | Experiment or dataset name            |

Refer to [internal/questdb/questdb.go](internal/questdb/questdb.go) for full schema logic.

---

## Example

Suppose you performed several audio experiments, each producing:

- A log events CSV (one row per audio run)
- Several per-frame audio CSVs
- Matched transcript CSVs

To import and index all data into QuestDB:

```bash
./mt_processaudio \
  -log-event-file ./logs/my_experiment_logs.csv \
  -audio-data-file-dir ./audio_frames/ \
  -transcript-file-dir ./transcripts/ \
  -name experiment_nov2025
```

Now, you can analyze both low-level audio features and high-level transcripted words in QuestDB or
connect to visualization tools such as Grafana. A dashboard to view the data is
available [here](https://github.com/olegbilovus/MT_ProcessPKTs/blob/main/Grafana_Dashboard.json)

---

## License

MIT License — see [LICENSE](LICENSE).

