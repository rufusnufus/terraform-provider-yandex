---
subcategory: "Managed Service for YDB"
page_title: "Yandex: yandex_ydb_database_iam_binding"
description: |-
  Allows management of a single IAM binding for a [Managed service for YDB](https://cloud.yandex.com/docs/ydb/).
---


# yandex_ydb_database_iam_binding




Allows creation and management of a single binding within IAM policy for an existing Managed YDB Database instance.

## Example usage

```terraform
resource "yandex_ydb_database_serverless" "database1" {
  name      = "test-ydb-serverless"
  folder_id = data.yandex_resourcemanager_folder.test_folder.id
}

resource "yandex_ydb_database_iam_binding" "viewer" {
  database_id = yandex_ydb_database_serverless.database1.id
  role        = "ydb.viewer"

  members = [
    "userAccount:foo_user_id",
  ]
}
```

## Argument Reference

The following arguments are supported:

* `database_id` - (Required) The [Managed Service YDB instance](https://cloud.yandex.com/docs/ydb/) Database ID to apply a binding to.

* `role` - (Required) The role that should be applied. See [roles](https://cloud.yandex.com/docs/ydb/security/).

* `members` - (Required) Identities that will be granted the privilege in `role`. Each entry can have one of the following values:
  * **userAccount:{user_id}**: A unique user ID that represents a specific Yandex account.
  * **serviceAccount:{service_account_id}**: A unique service account ID.
  * **federatedUser:{federated_user_id}:**: A unique saml federation user account ID.
  * **group:{group_id}**: A unique group ID.
  * **system:group:federation:{federation_id}:users**: All users in federation.
  * **system:group:organization:{organization_id}:users**: All users in organization.
  * **system:allAuthenticatedUsers**: All authenticated users.
  * **system:allUsers**: All users, including unauthenticated ones.

  Note: for more information about system groups, see the [documentation](https://cloud.yandex.com/docs/iam/concepts/access-control/system-group).

## Import

IAM binding imports use space-delimited identifiers; first the resource in question and then the role. These bindings can be imported using the `database_id` and role, e.g.

```
$ terraform import yandex_ydb_database_iam_binding.viewer "database_id ydb.viewer"
```