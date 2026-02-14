# Ajdad.net Family Tree Data

Open-source family tree data for [ajdad.net](https://ajdad.net).

This directory is planned to become a standalone public repository so that anyone can contribute family tree data via pull requests without needing access to the application codebase.

## Directory Structure

```
data/
  {country}/
    {family-name}.yaml
```

Example: `data/saudi-arabia/al-saud.yaml`

## YAML Format

Each file contains a single family tree with two top-level keys: `family` and `persons`.

```yaml
family:
  name: آل سعود
  type: aal                  # aal | aila | qabila | ashira | usra
  suffixMale: السعودي
  suffixFemale: السعودية
  hometown: الدرعية          # optional
  source: >-                 # optional — مصدر المعلومات
    مصدر المعلومات
  description: >-            # optional
    وصف العائلة
  pinned: true               # optional — pin to top of homepage
  visibility: public         # public | private

persons:
  - id: 3fa85f64-5717-4562-b3fc-2c963f66afa6
    name: مانع
    sex: male
    nickname: المريدي        # optional — اللقب (prefix title)
    kunya: مؤسس الدرعية     # optional — التعريف (postfix identifier)
    birthdate: "1400"        # optional — YYYY format, must be quoted

  - id: 9c1b5b5a-8a0e-4e3b-b9a2-6f1c3d7e8f90
    name: ربيعة
    sex: male
    fatherId: 3fa85f64-5717-4562-b3fc-2c963f66afa6
    motherId: ...            # optional — references another person's id

  - id: 7b2a9f31-4e68-4c0a-a512-8d3e5f6a7b9c
    name: فهد
    sex: male
    nickname: الملك          # prefix title
    kunya: الخامس            # postfix identifier
    fatherId: ...
```

## Person Fields

| Field | Required | Description |
|-------|----------|-------------|
| `id` | yes | UUIDv4 — becomes the Firestore document ID. Generate with `uuidgen` or any UUID tool. |
| `name` | yes | First name only (e.g. "فهد"). Do not include "بن [father]" — the parent relationship is expressed via `fatherId`. |
| `sex` | yes | `male` or `female` |
| `nickname` | no | اللقب — prefix title displayed before the name (e.g. "الملك", "الإمام", "ولي العهد") |
| `kunya` | no | التعريف — postfix identifier displayed after the name (e.g. "الخامس", "المؤسس", "الأول") |
| `birthdate` | no | Year in YYYY format, must be quoted (e.g. `"1921"`) |
| `fatherId` | no | References another person's `id` in the same file |
| `motherId` | no | References another person's `id` in the same file |

**`nickname` + `kunya` example:** For الملك فهد الخامس, set `nickname: الملك` and `kunya: الخامس`. The app renders: `nickname` + `name` + `kunya`.

## Rules

- **Topological order**: parents must appear before their children in the `persons` list.
- **`fatherId` / `motherId`** reference another person's `id` within the same file.
- **`family.type`** must be one of:
  | Value | Arabic |
  |-------|--------|
  | `aal` | آل |
  | `aila` | عائلة |
  | `qabila` | قبيلة |
  | `ashira` | عشيرة |
  | `usra` | أسرة |

## Import Modes

When imported via the admin panel at `/admin/import`:

- **No `family.id`** in the YAML = **create mode** (new family + all persons)
- **`family.id` present** = **update mode** (updates existing family, creates new persons not found in Firestore)

## Contributing

1. Create a YAML file under the appropriate `data/{country}/` directory
2. Generate real UUIDv4 IDs for each person (do not use placeholder or hand-written IDs)
3. Ensure parents are listed before children
4. Submit a pull request

All data contributed here is public. Do not include private or sensitive personal information about living individuals without their consent.

## License

This data is provided as-is for educational and genealogical purposes.
