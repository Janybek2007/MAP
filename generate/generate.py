import hashlib
import json


def normalize_lat_lng(lat, lon):
    """
    Возвращает координаты в формате (lat, lng).
    Авто-фиксит случаи, когда значения пришли как (lng, lat).
    """
    lat = float(lat)
    lon = float(lon)

    # Явно перепутано по общим гео-диапазонам
    if abs(lat) > 90 and abs(lon) <= 90:
        return lon, lat

    # Эвристика под Кыргызстан: lat ~ 39..44, lng ~ 69..81
    if 69 <= lat <= 81 and 39 <= lon <= 44:
        return lon, lat

    return lat, lon


def flatten_items(items):
    """
    Рекурсивно превращает вложенные списки в один плоский список.
    """
    flat_list = []

    for item in items:
        if isinstance(item, list):
            flat_list.extend(flatten_items(item))
        else:
            flat_list.append(item)

    return flat_list


def generate_hash_part(value):
    raw = (value or "").strip().lower()
    return hashlib.md5(raw.encode("utf-8")).hexdigest()


def generate_hash_id(name, category, address):
    """
    Стабильный hash_id из name:hash(category):hash(address).
    """
    raw = f"{(name or '').strip().lower()}:{generate_hash_part(category)}:{generate_hash_part(address)}"

    return hashlib.md5(raw.strip().lower().encode("utf-8")).hexdigest()


def generate_locations(
    json_data,
    category_type,
    category_name,
    category_display=None,
    child_category_display=None,
):
    """
    Парсит JSON, выпрямляет вложенные массивы и извлекает координаты.
    """

    if isinstance(json_data, str):
        try:
            with open(json_data, encoding="utf-8") as f:
                data = json.load(f)
        except Exception as e:
            print(f"Ошибка при чтении {json_data}: {e}")
            return []
    else:
        data = json_data

    raw_items = data.get("items", [])
    items = flatten_items(raw_items)

    result = []
    seen_coords = set()

    for item in items:
        if not isinstance(item, dict):
            continue

        lat = item.get("lat")
        lon = item.get("lon")

        # Если координаты внутри point
        if lat is None or lon is None:
            point = item.get("point", {})

            if isinstance(point, dict):
                lat = point.get("lat")
                lon = point.get("lon")

        if lat is None or lon is None:
            continue

        lat, lon = normalize_lat_lng(lat, lon)

        coord_key = (
            round(lat, 6),
            round(lon, 6),
        )

        # Удаление дублей
        if coord_key in seen_coords:
            continue

        seen_coords.add(coord_key)

        name = item.get("name")

        address_name = item.get("address_name")
        # Стабильный hash id
        hid = generate_hash_id(
            name=name,
            category=category_type,
            address=address_name,
        )

        result.append(
            {
                "hid": hid,
                "name": name,
                "address": address_name,
                "category": category_type,
                "child_category": category_name,
                "category_display": category_display,
                "child_category_display": child_category_display,
                "lat": lat,
                "lng": lon,
            }
        )

    return result


# --- Основной цикл ---

locations_config = [
    # ("частные-клиники.json", "private", "private_clinics"),
    # ("кардиоцентры.json", "state_medical", "state_medical_cardiology"),
    # ("цсм.json", "state_medical", "state_medical_csm"),
    # ("нхц.json", "state_medical", "state_medical_nch"),
    # ("больницы.json", "state_medical", "state_medical_hospitals"),
    # ("филиалы-бонецкий.json", "bonetsky", "bonetsky_branches"),
    ("экспресс.json", ("rival", "Конкуренты"), ("rival_express", "Экспресс")),
    ("сапат.json", ("rival", "Конкуренты"), ("rival_sapat", "Сапат")),
    ("аквалаб.json", ("rival", "Конкуренты"), ("rival_akvalab", "Аквалаб")),
    ("евролаб.json", ("rival", "Конкуренты"), ("rival_evrolab", "Евролаб")),
]

all_locations = []
global_seen = set()
source_stats = []

for json_path, category_type, category_name in locations_config:
    print(f"Обработка: {json_path}")

    category_value = category_type
    category_display = None
    if isinstance(category_type, tuple):
        category_value, category_display = category_type

    child_value = category_name
    child_display = None
    if isinstance(category_name, tuple):
        child_value, child_display = category_name

    data = generate_locations(
        json_path,
        category_value,
        child_value,
        category_display=category_display,
        child_category_display=child_display,
    )
    source_total = len(data)
    source_added = 0

    for item in data:
        coord_key = (
            round(float(item["lat"]), 6),
            round(float(item["lng"]), 6),
        )

        # Глобальное удаление дублей
        if coord_key in global_seen:
            continue

        global_seen.add(coord_key)
        all_locations.append(item)
        source_added += 1

    source_stats.append((json_path, source_total, source_added))

# Push в существующий result.json (без перезаписи массива locations)
existing = {"locations": []}
try:
    with open("result.json", encoding="utf-8") as f:
        loaded = json.load(f)
        if isinstance(loaded, dict) and isinstance(loaded.get("locations"), list):
            existing = loaded
except FileNotFoundError:
    pass
except Exception:
    pass

existing_locations = existing.get("locations", [])
existing_hids = set()
for row in existing_locations:
    if isinstance(row, dict):
        hid = row.get("hid")
        if hid:
            existing_hids.add(str(hid))

added = 0
for item in all_locations:
    hid = str(item.get("hid") or "")
    if not hid or hid in existing_hids:
        continue
    existing_locations.append(item)
    existing_hids.add(hid)
    added += 1

existing["locations"] = existing_locations

with open("result.json", "w", encoding="utf-8") as f:
    json.dump(existing, f, indent=2, ensure_ascii=False)

print(f"Готово! Добавлено: {added}")
print(f"Итого в файле: {len(existing_locations)}")
print("Статистика по источникам:")
for name, total, unique_added in source_stats:
    print(f"- {name}: нашло {total}, добавилось {unique_added}")
