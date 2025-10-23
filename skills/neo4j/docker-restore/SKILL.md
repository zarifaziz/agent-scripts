---
name: docker-restore
description: Restore Neo4j database backups from Google Cloud Storage to a local Docker container. Use when the user asks to restore a Neo4j backup, mentions gsutil backup paths, or needs to restore from gs://neo4j-backups-dev/resources/.
allowed-tools:
  - bash
metadata:
  version: "1.0"
---

# Neo4j Docker Restore

## Overview
This skill restores Neo4j database backups from Google Cloud Storage to a local Docker Neo4j instance. It handles the complete workflow: downloading from GCS, copying to container, stopping the database, restoring, and restarting.

## Prerequisites
- Docker container named `resources-db` running Neo4j
- gsutil CLI installed and configured
- Neo4j credentials: username `neo4j`, password `password`
- Backup files in format: `neo4j-YYYY-MM-DDTHH-MM-SS.backup`

## Restoration Workflow

### Step 1: Download Backup from GCS
```bash
gsutil cp 'gs://neo4j-backups-dev/resources/<backup-file-name>' .
```

Example:
```bash
gsutil cp 'gs://neo4j-backups-dev/resources/neo4j-2025-10-17T12-03-02.backup' .
```

### Step 2: Copy to Docker Container
```bash
docker cp "./<backup-file-name>" resources-db:/var/lib/neo4j/import/<backup-file-name>
```

### Step 3: Stop the Database
```bash
docker exec resources-db cypher-shell -u neo4j -p password -d system "STOP DATABASE neo4j"
```

### Step 4: Restore the Backup
```bash
docker exec resources-db neo4j-admin database restore --overwrite-destination=true --from-path=/var/lib/neo4j/import/<backup-file-name> neo4j
```

This command will output restoration progress and create a metadata script at `/data/scripts/neo4j/restore_metadata.cypher`.

### Step 5: Start the Database
```bash
docker exec resources-db cypher-shell -u neo4j -p password -d system "START DATABASE neo4j"
```

### Step 6: Execute Metadata Restoration
```bash
docker exec resources-db cypher-shell -u neo4j -p password -d neo4j -f /data/scripts/neo4j/restore_metadata.cypher --param 'database => "neo4j"'
```

### Step 7: Cleanup
```bash
docker exec resources-db rm /var/lib/neo4j/import/<backup-file-name>
rm <backup-file-name>
```

## Complete Example

Given a backup URL `gs://neo4j-backups-dev/resources/neo4j-2025-10-17T12-03-02.backup`:

```bash
cd /path/to/project

gsutil cp 'gs://neo4j-backups-dev/resources/neo4j-2025-10-17T12-03-02.backup' .

docker cp "./neo4j-2025-10-17T12-03-02.backup" resources-db:/var/lib/neo4j/import/neo4j-2025-10-17T12-03-02.backup

docker exec resources-db cypher-shell -u neo4j -p password -d system "STOP DATABASE neo4j"

docker exec resources-db neo4j-admin database restore --overwrite-destination=true --from-path=/var/lib/neo4j/import/neo4j-2025-10-17T12-03-02.backup neo4j

docker exec resources-db cypher-shell -u neo4j -p password -d system "START DATABASE neo4j"

docker exec resources-db cypher-shell -u neo4j -p password -d neo4j -f /data/scripts/neo4j/restore_metadata.cypher --param 'database => "neo4j"'

docker exec resources-db rm /var/lib/neo4j/import/neo4j-2025-10-17T12-03-02.backup

rm neo4j-2025-10-17T12-03-02.backup
```

## Key Points

1. **Always stop the database** before restoration
2. **Metadata restoration is required** after restore completes
3. **Use the exact parameter format** for metadata script: `--param 'database => "neo4j"'`
4. **Cleanup both locations**: container import directory and local filesystem
5. **Container paths are fixed**: Import directory is always `/var/lib/neo4j/import/`

## Common Issues

**Issue**: "Expected parameter(s): database" when running metadata script
**Solution**: Use the exact format: `--param 'database => "neo4j"'` with the fat arrow `=>`

**Issue**: Download interrupted due to multiprocessing on macOS
**Solution**: Add `-o "GSUtil:parallel_process_count=1"` to gsutil command

**Issue**: Database won't stop
**Solution**: Check if database exists: `docker exec resources-db cypher-shell -u neo4j -p password -d system "SHOW DATABASES"`

## Automation Script Reference

The project includes `restore.sh` which automates this process. It:
- Finds the latest backup for a given database name
- Downloads and restores automatically
- Handles cleanup

To use the automation script:
```bash
./restore.sh
```

The script targets the `neo4j` database by default and finds the most recent backup matching `neo4j-*` pattern.
