# Remote repository local backup script

This is the code for backing up files stored in a remote repository configured with [`rclone`](https://rclone.org/) to local storage.

## Setup

Download and install [`rclone`](https://rclone.org/)

## Usage

Run

```bash
./remote_repo_backup --dest=$LOCAL_STORAGE_PATH
```

where `$LOCAL_STORAGE_PATH` is the absolute path to the intended local destination of the backup.

The script only copies files that are new and overwrites otherwise identical files that have been changed, comparing those in the source and destination. It utilizes `rclone`'s [`copy` command](https://rclone.org/commands/rclone_copy/).

The script also creates and maintains a `metadata.json` file at the backup destination. That file includes the following information for each invocation:

- when it started
- when it completed
- the number of files backed up
- the total size of the backup
- the number of non-fatal errors encountered

Invocations of the script will also use this file, if it exists, to filter for new files in the remote repository.

### Flags

1. `dest`: `string`, required: The absolute path to the destination of the backup. e.g. `Volumes/some_external_drive/backup`
2. `source`: `string`, default=`"google-drive"`: The label of the `rclone` remote to be backed up
3. `copyAll`: `boolean` default=`false`: Whether to copy all of the files from the source to the destination. When false, only the files changed since the last invocation are copied

## Development

The script lives in `backup.go`. Run `go build` to create a new executable after making changes.
