# Skills

When you're asked to use 'X' skill, look for the skill documentation inside ~/.opencode/skills/ folders and subfolders.
The Available skills are (but not limited to):

## Custom User Tools

Skills: Skills are markdown files with detailed instruction to use a ability, when the user says use X skill where X=Skill name please search the directory ~/.opencode/skills/<name> and read the SKILL.md file from it. For example to use 'project-oracle' skill you'll look into ~/.opencode/skills/project-oracle/SKILL.md and follow the guide there

## Available skills are (but not limited to):

`cypher-safe: Execute Cypher queries against Neo4j databases. Use when running read queries, testing write operations, or working with Neo4j databases.
db-oracle: Consult the all-knowing Neo4j Oracle for medium/large Cypher queries. The Oracle knows the entire database schema, all Go code interfacing with Neo4j, and every query pattern ever written.
docker-restore: Restore Neo4j database backups from Google Cloud Storage to a local Docker container. Use when the user asks to restore a Neo4j backup, mentions gsutil backup paths, or needs to restore from gs://neo4j-backups-dev/resources/.
inspect-resource-question: Extract and inspect AI-generated questions from lesson plans or worksheet plans. Formats quillDocument questions into readable plaintext.
piper-tts: Use piper-say TTS tool to provide audio feedback when completing code implementation tasks. Primarily uses English (piper-say) unless user explicitly requests Nepali (piper-sayn).
psql-safe: Execute SQL queries against PostgreSQL/CockroachDB databases. Use when running read queries, testing write operations, or working with CockroachDB databases.
reverse-graphql-input: Reverse engineer GraphQL input from a lesson plan or worksheet plan ID/link. Extracts plan data and reconstructs the original input JSON used to create it.`

# Accesibility Notice

I'm a blind developer so please use `piper-say`utility via piper-tts skill to inform me with one liner summary when you complete your task (put at the end of todo list wrap)
