# Lexorank

A production-ready Go implementation of the Lexorank sort-key system — originally designed by Atlassian to enable infinitely re-orderable lists with minimal rebalancing.

Lexorank allows you to insert new items between existing ones, and only rebalances when you truly run out of space. This library gives you a `Key` type and a `ReorderableList` utility to manage ordered collections safely and efficiently.

> Lexorank is used in [Storyden](https://storyden.org) to power drag-and-drop reordering of tree-structured or table-structured content. It handles thousands of inserts, normalizations, and concurrent reorders in production environments.

---

## Features

- Lexorank `Key`: Stable, lexicographically ordered string key
- Insert / Append / Prepend: Add new items without needing to re-fetch the entire list
- Partial rebalancing: If there's no room between two keys, rebalance just the nearby items
- Full normalization: Optionally normalize the entire list with evenly spaced keys
- Key generation with precision limit: Tells you to rebalance when key bounds are hit
- `Reorderable` interface: Integrate with your own data types

---

## Usage Pattern 1: Minimal context insert

If you're inserting an item and you already know the IDs (or keys) of the adjacent items, you can do the following:

```go
midKey, ok := leftKey.Between(rightKey)
if !ok {
    // No possible key between left and right - run a rebalance of the set
}
```

This avoids loading the full list and is ideal for quick, isolated inserts.

---

## Usage Pattern 2: List-based insertion with rebalancing

If you can load the full list of siblings (e.g., all items in a table, or children of a parent), you can use the `ReorderableList` utility:

```go
list := lexorank.ReorderableList(<yourdata>)
newKey := list.Insert(3) // Insert at position 3
```

This approach:

- Uses `.Between()` under the hood
- If needed, rebalances a small section of the list
- If still no space is available, performs a `.Normalise()` (safe for up to hundreds of thousands of items)

You can also manually normalise (distribute all keys evenly across a set)

```go
list := lexorank.ReorderableList(<yourdata>)
list.Normalise(3)
// write `list` back to your DB
```

This library does not implement any adapters or persistence, so you are responsible for writing back the changes to your `ReorderableList` instance to your database.

## Rebalancing and Precision

The current key character set is 75 characters (0-z ASCII) and the key length is 6 characters, which gives you:

- Approximately 177 billion unique keys
- Around 400,000 worst-case inserts between two keys before a rebalance

This makes Lexorank quite efficient for large document spaces with lots of drag/drop/move operations.

> If you run into these limits and require a configurable limit, open an issue or PR!

## Buckets

The library retains buckets internally, but they are currently not used by the underlying algorithm. Buckets were originally part of Atlassian's implementation to allow sharded normalization across large datasets. If you need to change the bucket, you can simply mutate the first character of a key.
