import hashlib
import json
from pathlib import Path

BASE_DIR = Path(__file__).parent

SOURCE_FILES = [
    BASE_DIR / "ГОС_финал.json",
    BASE_DIR / "Частные клиники_финал.json",
    BASE_DIR / "Филиалы бонецкого_финал.json",
]

RESULT_PATH = BASE_DIR / "result.json"


def hash_part(value: str | None) -> str:
    raw = (value or "").strip().lower()
    return hashlib.md5(raw.encode("utf-8")).hexdigest()


def hash_name(name: str | None, category: str | None, address: str | None) -> str:
    raw = f"{(name or '').strip().lower()}:{hash_part(category)}:{hash_part(address)}"
    return hashlib.md5(raw.encode("utf-8")).hexdigest()


def to_bool_partnership(value) -> bool:
    if isinstance(value, bool):
        return value
    if value is None:
        return False
    normalized = str(value).strip().lower()
    if normalized in {"да", "yes", "true", "1"}:
        return True
    if normalized in {"нет", "no", "false", "0", ""}:
        return False
    return False


def load_json_array(path: Path) -> list[dict]:
    if not path.exists():
        return []
    with path.open(encoding="utf-8") as f:
        data = json.load(f)
    if isinstance(data, list):
        return [row for row in data if isinstance(row, dict)]
    return []


def load_result() -> dict:
    if not RESULT_PATH.exists():
        return {"locations": []}
    try:
        with RESULT_PATH.open(encoding="utf-8") as f:
            data = json.load(f)
        if isinstance(data, dict) and isinstance(data.get("locations"), list):
            return data
    except Exception:
        pass
    return {"locations": []}


def main() -> None:
    result = load_result()
    locations = result.get("locations", [])

    existing_hids = set()
    for row in locations:
        if isinstance(row, dict):
            hid = row.get("hid")
            if hid:
                existing_hids.add(str(hid))

    added = 0
    source_stats = []
    for source_path in SOURCE_FILES:
        rows = load_json_array(source_path)
        source_total = len(rows)
        source_added = 0
        for row in rows:
            name = row.get("name")
            hid = hash_name(name, row.get("category"), row.get("address"))
            if hid in existing_hids:
                continue

            item = dict(row)
            item["hid"] = hid
            item["is_partnerships"] = to_bool_partnership(item.get("is_partnerships"))
            item["address"] = item.get("address") or None
            item["manager"] = item.get("manager") or None
            locations.append(item)
            existing_hids.add(hid)
            added += 1
            source_added += 1
        source_stats.append((source_path.name, source_total, source_added))

    result["locations"] = locations
    with RESULT_PATH.open("w", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False, indent=2)

    print(f"saved: {RESULT_PATH}")
    print(f"added: {added}")
    print(f"total: {len(locations)}")
    print("Статистика по источникам:")
    for name, total_rows, added_rows in source_stats:
        print(f"- {name}: нашло {total_rows}, добавилось {added_rows}")


if __name__ == "__main__":
    main()
