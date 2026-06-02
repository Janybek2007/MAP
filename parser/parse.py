import argparse
import json
import re
from pathlib import Path

from slugify import slugify

BASE_DIR = Path(__file__).parent
SOURCE_GOS_PATH = BASE_DIR / "ГОС.json"
SOURCE_BONETSKY_PATH = BASE_DIR / "Филиалы Бонецкого.json"
GENERATE_DIR = BASE_DIR.parent / "generate"
OUTPUT_PUBLIC = GENERATE_DIR / "ГОС_финал.json"
OUTPUT_PRIVATE = GENERATE_DIR / "Частные клиники_финал.json"
OUTPUT_BONETSKY = GENERATE_DIR / "Филиалы бонецкого_финал.json"

COORD_RE = re.compile(r"(-?\d+(?:\.\d+)?)")


def parse_lat_lng(value: str) -> tuple[float | None, float | None]:
    if not value:
        return None, None

    parts = COORD_RE.findall(str(value))
    if len(parts) < 2:
        return None, None

    first = float(parts[0])
    second = float(parts[1])

    # В исходнике формат "LNG,LAT", возвращаем "LAT,LNG".
    lat = second
    lng = first
    return lat, lng


def normalize_address(value: str | None) -> str | None:
    if value is None:
        return None
    cleaned = str(value).replace("_", "").strip()
    return cleaned or None


def capitalize_first(value: str) -> str:
    value = (value or "").strip()
    if not value:
        return ""
    return value[0].upper() + value[1:]


def normalize_group(value: str) -> str:
    """
    Нормализует "Группа" для child_category:
    - "гинекология, урология" => "гинекология"
    """
    raw = (value or "").strip()
    if not raw:
        return ""
    primary = raw.split(",", 1)[0].strip()
    return primary or raw


def normalize_lat_lng(lat: float, lng: float) -> tuple[float, float]:
    """
    Возвращает координаты в формате (lat, lng).
    Авто-фиксит случаи, когда значения пришли как (lng, lat).
    """
    # Явно перепутано по общим гео-диапазонам
    if abs(lat) > 90 and abs(lng) <= 90:
        return lng, lat

    # Эвристика под Кыргызстан: lat ~ 39..44, lng ~ 69..81
    if 69 <= lat <= 81 and 39 <= lng <= 44:
        return lng, lat

    return lat, lng


def parse_coords_lat_lng(value: str) -> tuple[float | None, float | None]:
    if not value:
        return None, None

    parts = COORD_RE.findall(str(value))
    if len(parts) < 2:
        return None, None

    first = float(parts[0])
    second = float(parts[1])

    # В новом формате обычно "LAT, LNG", но иногда перепутано.
    lat, lng = normalize_lat_lng(first, second)
    return lat, lng


def make_item(row: dict) -> dict | None:
    lat, lng = parse_lat_lng(row.get("LAT,LNG", ""))
    name = (row.get("Наименование") or "").strip()
    if not name or lat is None or lng is None:
        return None

    address = normalize_address(row.get("Адрес"))

    raw_type = (row.get("ТИП гос / частный") or "").strip()
    raw_group = (row.get("Группа") or "").strip()
    raw_group_norm = normalize_group(raw_group)
    raw_partnerships = (row.get("Партнерства Да/Нет") or "").strip()
    raw_manager = (row.get("Менеджер") or "").strip()

    return {
        "name": name,
        "address": address,
        "lat": lat,
        "lng": lng,
        "category": slugify(raw_type, separator="_"),
        "child_category": slugify(raw_group_norm, separator="_"),
        "category_display": capitalize_first(raw_type),
        "child_category_display": capitalize_first(raw_group_norm),
        "is_partnerships": raw_partnerships,
        "manager": raw_manager,
    }


def process_gos() -> None:
    with SOURCE_GOS_PATH.open(encoding="utf-8") as file:
        rows = json.load(file)

    gos_items = []
    private_items = []

    for row in rows:
        if not isinstance(row, dict):
            continue
        item = make_item(row)
        if not item:
            continue

        category_src = (row.get("ТИП гос / частный") or "").strip().lower()
        if "гос" in category_src:
            gos_items.append(item)
        elif "част" in category_src:
            private_items.append(item)

    GENERATE_DIR.mkdir(parents=True, exist_ok=True)
    OUTPUT_PUBLIC.write_text(json.dumps(gos_items, ensure_ascii=False, indent=2), encoding="utf-8")
    OUTPUT_PRIVATE.write_text(json.dumps(private_items, ensure_ascii=False, indent=2), encoding="utf-8")

    print(f"saved: {OUTPUT_PUBLIC}")
    print(f"saved: {OUTPUT_PRIVATE}")
    print(f"gos: {len(gos_items)}")
    print(f"private: {len(private_items)}")


def make_bonetsky_item(row: dict) -> dict | None:
    coords = (row.get("Координаты") or "").strip()
    lat, lng = parse_coords_lat_lng(coords)
    name = (row.get("Именование") or "").strip()
    if not name or lat is None or lng is None:
        return None

    address = normalize_address(row.get("Адрес"))
    raw_type = (name.split()[:1] or [""])[0].strip()

    return {
        "name": name,
        "address": address,
        "lat": lat,
        "lng": lng,
        "type": slugify(raw_type, separator="_"),
        "type_display": raw_type,
        "category": "bonetsky",
        "child_category": "bonetsky",
        "category_display": "Филиалы бонецкого",
        "child_category_display": "Филиалы бонецкого",
    }


def process_bonetsky() -> None:
    with SOURCE_BONETSKY_PATH.open(encoding="utf-8") as file:
        rows = json.load(file)

    items = []
    for row in rows:
        if not isinstance(row, dict):
            continue
        item = make_bonetsky_item(row)
        if item:
            items.append(item)

    GENERATE_DIR.mkdir(parents=True, exist_ok=True)
    OUTPUT_BONETSKY.write_text(json.dumps(items, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"saved: {OUTPUT_BONETSKY}")
    print(f"bonetsky: {len(items)}")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Parse datasets into final JSON.")
    parser.add_argument("--gos", action="store_true", help="Process ГОС dataset.")
    parser.add_argument("--bonetsky", action="store_true", help="Process Бонецкий dataset.")
    return parser.parse_args()


def main() -> None:
    args = parse_args()

    # По умолчанию сохраняем прежнее поведение.
    if not args.gos and not args.bonetsky:
        process_gos()
        return

    if args.gos:
        process_gos()
    if args.bonetsky:
        process_bonetsky()


if __name__ == "__main__":
    main()
