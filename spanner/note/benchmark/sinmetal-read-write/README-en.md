# Spanner Benchmark by @sinmetal

tag["google-cloud-spanner"]

* The following is a benchmark for Google Cloud Spanner’s capabilities to READ/WRITE in multiple tables.
* The processes below are nothing special – we simply wanted to test if using an INSERT -> UPDATE -> UPDATE process would improve performance.
* It should be noted that the INSERT -> UPDATE -> UPDATE process was found to somewhat improve performance. 

## Procedures

* While using the same Tx, Insert into the five tables: Tweet, Operation, TweetDummy1, TweetDummy2, and TweetDummy3.


``` https://github.com/sinmetal/alminium_spanner/blob/8ec636214e22d6842d0951e5f0a8885a73bfec4c/tweet_store.go#L191
func (s *defaultTweetStore) InsertBench(ctx context.Context, id string) error {
        ctx, span := trace.StartSpan(ctx, "/tweet/insertbench")
        defer span.End()

        ml := []*spanner.Mutation{}
        now := time.Now()

        t := &Tweet{
                ID:         id,
                Content:    id,
                Favos:      []string{},
                CreatedAt:  now,
                UpdatedAt:  now,
                CommitedAt: spanner.CommitTimestamp,
        }
        tm, err := spanner.InsertStruct(s.TableName(), t)
        if err != nil {
                return err
        }
        ml = append(ml, tm)

        tom, err := NewOperationInsertMutation(uuid.New().String(), "INSERT", "", s.TableName(), t)
        if err != nil {
                return err
        }
        ml = append(ml, tom)

        for i := 1; i < 4; i++ {
                td := &TweetDummy{
                        ID:         id,
                        Content:    id,
                        Favos:      []string{},
                        CreatedAt:  now,
                        UpdatedAt:  now,
                        CommitedAt: spanner.CommitTimestamp,
                }
                tdm, err := spanner.InsertStruct(fmt.Sprintf("TweetDummy%d", i), td)
                if err != nil {
                        return err
                }
                ml = append(ml, tdm)
        }
        _, err = s.sc.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
                return txn.BufferWrite(ml)
        })

        return err
}
```

* Increment the count property of tweet. At the same time, acquire TweetDummy2 as well using the same Tx.

```
func (s *defaultTweetStore) Update(ctx context.Context, id string) error {
        ctx, span := trace.StartSpan(ctx, "/tweet/update")
        defer span.End()

        _, err := s.sc.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
                tr, err := txn.ReadRow(ctx, s.TableName(), spanner.Key{id}, []string{"Count"})
                if err != nil {
                        return err
                }
                _, err = txn.ReadRow(ctx, "TweetDummy2", spanner.Key{id}, []string{"Id"})
                if err != nil {
                        return err
                }

                var count int64
                if err := tr.ColumnByName("Count", &count); err != nil {
                        return err
                }
                count++
                cols := []string{"Id", "Count", "UpdatedAt", "CommitedAt"}

                return txn.BufferWrite([]*spanner.Mutation{
                        spanner.Update(s.TableName(), cols, []interface{}{id, count, time.Now(), spanner.CommitTimestamp}),
                })
        })

        return err
}
```

Repeat the processes above using 120 Goroutines.
We see that for one execution, the READ numbers are slightly higher, with the RW numbers adding up to 10 READs and 6 WRITEs. 

## Environment

### Region

* Set all regions to `asia-northeast1`

### Execution Environment

* The client is a standalone application on GKE.

* The source code can be found at https://github.com/sinmetal/alminium_spanner.

### GKE Specs

* Master Version: 1.10.7-gke.2
* Machine Type: A Nodepool which uses min1, max12 to auto-scale `n1-highcpu-8`

### Table

```
CREATE TABLE Tweet (
   Id STRING(MAX) NOT NULL,
   Author STRING(MAX) NOT NULL,
   CommitedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
   Content STRING(MAX) NOT NULL,
   Count INT64 NOT NULL,
   CreatedAt TIMESTAMP NOT NULL,
   Favos ARRAY<STRING(MAX)> NOT NULL,
   Sort INT64 NOT NULL,
   UpdatedAt TIMESTAMP NOT NULL,
) PRIMARY KEY (Id);

CREATE INDEX TweetSortAsc
ON Tweet (
        Sort
);

CREATE TABLE Operation (
        Id STRING(MAX) NOT NULL,
        VERB STRING(MAX) NOT NULL,
        TargetKey STRING(MAX) NOT NULL,
        TargetTable STRING(MAX) NOT NULL,
        Body BYTES(MAX),
        CommitedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
) PRIMARY KEY (Id);

CREATE INDEX OperationTargetKey
ON Operation (
TargetKey
);

CREATE INDEX OperationTargetTable
ON Operation (
TargetTable
);

CREATE TABLE TweetDummy1 (
   Id STRING(MAX) NOT NULL,
   Author STRING(MAX) NOT NULL,
   CommitedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
   Content STRING(MAX) NOT NULL,
   Count INT64 NOT NULL,
   CreatedAt TIMESTAMP NOT NULL,
   Favos ARRAY<STRING(MAX)> NOT NULL,
   Sort INT64 NOT NULL,
   UpdatedAt TIMESTAMP NOT NULL,
) PRIMARY KEY (Id);

CREATE INDEX TweetDummy1SortAsc
ON TweetDummy1 (
        Sort
);

CREATE UNIQUE INDEX TweetDummy1Content ON TweetDummy1(Content);

CREATE TABLE TweetDummy2 (
   Id STRING(MAX) NOT NULL,
   Author STRING(MAX) NOT NULL,
   CommitedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
   Content STRING(MAX) NOT NULL,
   Count INT64 NOT NULL,
   CreatedAt TIMESTAMP NOT NULL,
   Favos ARRAY<STRING(MAX)> NOT NULL,
   Sort INT64 NOT NULL,
   UpdatedAt TIMESTAMP NOT NULL,
) PRIMARY KEY (Id);

CREATE INDEX TweetDummy2SortAsc
ON TweetDummy2 (
        Sort
);

CREATE UNIQUE INDEX TweetDummy2Content ON TweetDummy2(Content);

CREATE TABLE TweetDummy3 (
   Id STRING(MAX) NOT NULL,
   Author STRING(MAX) NOT NULL,
   CommitedAt TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp=true),
   Content STRING(MAX) NOT NULL,
   Count INT64 NOT NULL,
   CreatedAt TIMESTAMP NOT NULL,
   Favos ARRAY<STRING(MAX)> NOT NULL,
   Sort INT64 NOT NULL,
   UpdatedAt TIMESTAMP NOT NULL,
) PRIMARY KEY (Id);

CREATE INDEX TweetDummy3SortAsc
ON TweetDummy3 (
        Sort
);

CREATE UNIQUE INDEX TweetDummy3Content ON TweetDummy3(Content);
```

## Report

### 11/07 12:00 (Node3, Row: All Table 20,065,380 cases, DB Size 41.17 GB)

* WRITE: 4000Tx/sec
* READ: 7000/sec

I found that at 9:45AM, the per-second processing speed decreased very briefly. However, the CPU usage for Spanner had also dropped so maybe the session had been closed and restored, or live migration was performed on GCE.
The Trace results showed there was one case where a lot of time was taken – this was due to the restoring of ReadWriteTx data.

![Stackdriver Monitoring](20181107-1200-node3-monitoring.png "Stackdriver Monitoring")
![Stackdriver Trace](20181107-1200-node3-trace.png "Stackdriver Trace")

It should be noted that five CPU cores were being used on the client side to run these processes.