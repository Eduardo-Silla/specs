---
menuTitle: Contributing
title: "Contributing to the Filecoin spec"
---

## Adding new sections

The specification is broken down into 5 levels (`#.#.#.#.#`). The L1 and L2 numbers in this sequence are determined by the first two directories extending from `/src/`; for example, `/src/systems/filecoin_mining/` resolves to `2.6.`.

The L3 number is generated by creating an additional directory within a L2 directory, containing its own appropriately formatted `index.md`. This new directory name must then be added to the `entries` field of the L2 `index.md` file, sequentially ordered as they are to be within the specification.

Further L4 and L5 subsections are added using the `##` and `###` headers respectively within a the content of a L3 section's `index.md` file.
