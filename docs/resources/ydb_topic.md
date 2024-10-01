---
subcategory: "Managed Service for YDB"
page_title: "Yandex: yandex_ydb_topic"
description: |-
  Get information about a Yandex YDB Topics.
---


# yandex_ydb_topic




Get information about a Yandex YDB Topics. For more information, see [the official documentation](https://cloud.yandex.ru/docs/ydb/concepts/#ydb).

## Example usage

```terraform
resource "yandex_ydb_database_serverless" "database_name" {
  name        = "database-name"
  location_id = "ru-central1"
}


resource "yandex_ydb_topic" "topic" {
  database_endpoint = yandex_ydb_database_serverless.database_name.ydb_full_endpoint
  name              = "topic-test"

  supported_codecs    = ["raw", "gzip"]
  partitions_count    = 1
  retention_period_ms = 2000000
  consumer {
    name                          = "consumer-name"
    supported_codecs              = ["raw", "gzip"]
    starting_message_timestamp_ms = 0
  }
}
```

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are exported:

* `name` - Topic name. Type: string, required. Default value: "".
* `database_endpoint` - YDB database endpoint. Types: string, required. Default value: "".
* `partitions_count` - Number of partitions. Types: integer, optional. Default value: 2.
* `retention_period_ms` - Data retention time. Types: integer, required. Default value: 86400000
* `supported_codecs` - Supported data encodings. Types: array[string]. Default value: ["gzip", "raw", "zstd"].
* `metering_mode` - Resource metering mode (`reserved_capacity` - based on the allocated resources or `request_units` - based on actual usage). This option applies to topics in serverless databases. Value type: String.
* `consumer` - Topic Readers. Types: array[consumer], optional. Default value: null.
* `partition_write_speed_kbps` - Maximum allowed write speed per partition. If a write speed for a given partition exceeds this value, the write speed will be capped. Value type: integer, default value: 1024 (1MB).

## Consumer data type description

* `name` - Reader's name. Type: string, required. Default value: "".
* `supported_codecs` - Supported data encodings. Types:Types: array[string], optional. Default value: ["gzip", "raw", "zstd"].
* `starting_message_timestamp_ms` - Timestamp in UNIX timestamp format from which the reader will start reading data. Type: integer, optional. Default value: 0.
* `important` - Defines an important consumer. No data will be deleted from the topic until all the important consumers read them. Value type: boolean, default value: false.