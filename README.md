# Graylog Archiver

<a href="http://hits.dwyl.com/minhpq331/graylog-archiver"><img alt="hits" src="https://hits.dwyl.com/minhpq331/graylog-archiver.svg?style=flat-square"></a> 

A command-line tool written in Go to automate the process of creating backups (snapshots) for OpenSearch indices (expecially for Graylog). The tool ensures no duplicate snapshots are created, supports analyzing index data for timestamp ranges, and allows filtering indices based on patterns and recency.

## Features

- Automatically discovers indices matching a specified pattern.
- Archives indices older than a specified number of recent indices.
- Prevents duplicate snapshots by checking existing snapshots in the repository.
- Supports optional analysis of indices to include min and max timestamps in the snapshot name.

## Prerequisites

- **Go**: Ensure Go is installed on your system.
- **OpenSearch Cluster**: An operational OpenSearch cluster with a configured snapshot repository.

## Installation

Just download the binary from [release page](https://github.com/minhpq331/graylog-archiver/releases) and you're good to go!

## Build it yourself

1. Clone the repository:
```bash
git clone https://github.com/minhpq331/graylog-archiver.git
cd graylog-archiver
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the executable:
```bash
go build -o graylog-archiver
```

## Usage

Run the CLI with the following arguments:

./graylog-archiver --pattern <pattern> --url <opensearch_url> --bypass <num> --repo <repository_name> [--analyze]

### Arguments

| Argument    | Description                                                   | Required | Example                 |
|-------------|---------------------------------------------------------------|----------|-------------------------|
| `--pattern` | The pattern for matching indices (e.g., uat_*).               | Yes      | `uat_*`                |
| `--url`     | The URL of the OpenSearch cluster.                            | Yes      | `http://localhost:9200` |
| `--bypass`  | Number of recent indices to skip from archiving.              | Yes      | `3`                     |
| `--repo`    | The name of the snapshot repository in OpenSearch.            | Yes      | `s3_backup_repo`        |
| `--analyze` | Enable analysis of min/max timestamps in index data (default: disabled). | No       |                         |

### Example

To back up all indices matching `uat_*`, skipping the latest 3, and using the repository s3_backup_repo:

```bash
./graylog-archiver --pattern "uat_*" --url http://localhost:9200 --bypass 3 --repo s3_backup_repo
```

To include timestamp analysis in the snapshot names:

```bash
./graylog-archiver --pattern "uat_*" --url http://localhost:9200 --bypass 3 --repo s3_backup_repo --analyze
```

**Snapshot Name Format**

The snapshot name follows this pattern:
- Without analysis: <index_name>
- With analysis: <index_name>.<from_timestamp>.<to_timestamp>

Example:
- Without analysis: uat_1
- With analysis: uat_1.20241101-1200.20241122-1230

## How It Works

1.	Fetch Indices: The tool fetches indices matching the specified pattern and sorts them from latest to oldest.
2.	Filter Indices: It skips the specified number of recent indices.
3.	Check for Duplicate Snapshots: Before creating a snapshot, the tool checks if a snapshot with the same name already exists.
4.	Analyze Timestamps (Optional): If enabled, the tool queries the index for the min and max @timestamp values.
5.	Create Snapshot: A snapshot is created in the specified repository for each eligible index.

## Deployment

Setting Up as a Cron Job

To run the tool automatically, add it to a cron job. For example:

```
0 2 * * * /path/to/graylog-archiver --pattern "uat_*" --url http://localhost:9200 --bypass 3 --repo s3_backup_repo --analyze >> /var/log/graylog-archiver.log 2>&1
```

This runs the tool daily at 2:00 AM and logs output to /var/log/graylog-archiver.log.

## License

This project is licensed under the Apache 2.0 License. See the LICENSE file for details.

## Contributing

Contributions are welcome! Please open issues or pull requests to contribute to this project.