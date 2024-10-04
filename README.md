# Lexorank

A Go implementation of the Lexorank sorting type.

---

This library provides a `Key` type to represent a Lexorank string. It also provides a `Reorderable` interface which you can satisfy in your own type which can then be stored in a `ReorderableList` on which `Insert`, `Append` and `Prepend` can be used to generate new keys as well as re-balance the underlying list if necessary.

In its current state, it's not 100% test coverage yet and considered a work in progress, but the current tests cover the happy paths. Edge cases include inserting between two keys which have no midpoint. Buckets are currently unused since they are primarily for performing a live-rebalance on a database.

## Usage pattern 1: inserting without querying the full range

When you want to insert or move an item, query the "surrounding" items of the target index (the item before and the item afterwards) get their keys and call `a.Between(b)` to get the sort key for placing an item in the middle. If this function returns `(nil, false)` then the two keys are adjacent and the table must be partially or fully re-balanced.

## Usage pattern 2: query the range, mutate the list

If you are fine with querying the full list of affected rows (perhaps grouped by some contextual parent container) you can use the `ReorderableList` type to perform insertion, append and prepend operations. This approach provides partial re-balancing where the items in the list will be assigned new Lexorank sort keys in order to normalise the range before returning a new key.

## Contributing

The implementation is not necessarily bulletproof right now, but the prior art I could find for the Go language were incomplete. If you have improvements, please open an issue or submit a pull request.
