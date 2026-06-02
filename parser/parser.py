"""
Население (population) в итоговых JSON строится из 3 источников:

1) Города (CITY_POPULATION_OVERRIDES)
   - Бишкек: 1 321 900
   - Ош: 473 500
   - Токмок: 71 443

2) Области (REGION_POPULATION_OVERRIDES, данные на 2025 год)
   - Ошская: 1 416 700
   - Джалал-Абадская: 1 358 500
   - Чуйская: 971 300
   - Баткенская: 594 700
   - Иссык-Кульская: 549 800
   - Нарынская: 314 900
   - Таласская: 280 500

3) Районы (kyrgyzstan_data)
   - population берётся напрямую из списка районов и не пересчитывается.

Важно:
- Бишкек и Ош статистически считаются отдельно (города республиканского значения).
- В UI города могут быть привязаны к областям через CITY_REGION_OVERRIDES, но это не означает
  суммирование населения города в население области (области берутся по overrides).
"""

import hashlib
import json
from pathlib import Path

kyrgyzstan_data = [
    {"область": "Чуйская", "город": None, "район": "Аламудунский", "население": 194300},
    {"область": "Чуйская", "город": None, "район": "Жайылский", "население": 115700},
    {"область": "Чуйская", "город": None, "район": "Кеминский", "население": 49400},
    {"область": "Чуйская", "город": None, "район": "Московский", "население": 106300},
    {"область": "Чуйская", "город": None, "район": "Панфиловский", "население": 49000},
    {"область": "Чуйская", "город": None, "район": "Сокулукский", "население": 255200},
    {"область": "Чуйская", "город": None, "район": "Чуйский", "население": 55800},
    {"область": "Чуйская", "город": None, "район": "Ысык-Атинский", "население": 159800},
    {"область": "Ошская", "город": None, "район": "Алайский", "население": 91800},
    {"область": "Ошская", "город": None, "район": "Араванский", "население": 139100},
    {"область": "Ошская", "город": None, "район": "Кара-Кульджинский", "население": 104500},
    {"область": "Ошская", "город": None, "район": "Кара-Сууский", "население": 465600},
    {"область": "Ошская", "город": None, "район": "Ноокатский", "население": 319200},
    {"область": "Ошская", "город": None, "район": "Узгенский", "население": 298400},
    {"область": "Ошская", "город": None, "район": "Чон-Алайский", "население": 34800},
    {"область": "Джалал-Абадская", "город": None, "район": "Аксыйский", "население": 143200},
    {"область": "Джалал-Абадская", "город": None, "район": "Ала-Букинский", "население": 111300},
    {"область": "Джалал-Абадская", "город": None, "район": "Базар-Коргонский", "население": 187800},
    {"область": "Джалал-Абадская", "город": None, "район": "Ноокенский", "население": 141900},
    {"область": "Джалал-Абадская", "город": None, "район": "Сузакский", "население": 319800},
    {"область": "Джалал-Абадская", "город": None, "район": "Тогуз-Тороуский", "население": 26500},
    {"область": "Джалал-Абадская", "город": None, "район": "Токтогульский", "население": 105900},
    {"область": "Джалал-Абадская", "город": None, "район": "Чаткальский", "население": 31500},
    {"область": "Иссык-Кульская", "город": None, "район": "Ак-Суйский", "население": 71200},
    {"область": "Иссык-Кульская", "город": None, "район": "Жети-Огузский", "население": 96200},
    {"область": "Иссык-Кульская", "город": None, "район": "Иссык-Кульский", "население": 87800},
    {"область": "Иссык-Кульская", "город": None, "район": "Тонский", "население": 57100},
    {"область": "Иссык-Кульская", "город": None, "район": "Тюпский", "население": 69800},
    {"область": "Нарынская", "город": None, "район": "Ак-Талинский", "население": 33900},
    {"область": "Нарынская", "город": None, "район": "Ат-Башинский", "население": 57300},
    {"область": "Нарынская", "город": None, "район": "Жумгальский", "население": 48400},
    {"область": "Нарынская", "город": None, "район": "Кочкорский", "население": 70100},
    {"область": "Нарынская", "город": None, "район": "Нарынский", "население": 53500},
    {"область": "Баткенская", "город": None, "район": "Баткенский", "население": 97300},
    {"область": "Баткенская", "город": None, "район": "Кадамжайский", "население": 210500},
    {"область": "Баткенская", "город": None, "район": "Лейлекский", "население": 154200},
    {"область": "Таласская", "город": None, "район": "Айтматовский", "население": 72400},
    {"область": "Таласская", "город": None, "район": "Бакай-Атинский", "население": 57800},
    {"область": "Таласская", "город": None, "район": "Манасский", "население": 39500},
    {"область": "Таласская", "город": None, "район": "Таласский", "население": 79100},
]

BASE_DIR = Path(__file__).parent
SRC_DIR = BASE_DIR / "kgz_admin_boundaries.geojson"
FALLBACK_DISTRICTS_PATH = BASE_DIR / "4_cities_districts.json"
DATA_DIR = BASE_DIR.parent / "data"
BISHKEK_DISTRICTS = {"Ленинский", "Октябрьский", "Первомайский", "Свердловский"}
CITY_POPULATION_OVERRIDES = {
    # НСК КР: "Численность постоянного населения на начало года"
    # (stat.gov.kg Open Data, значения 2025 года переведены из тыс. человек в человек)
    "Бишкек": 1321900,
    "Ош": 473500,
    "Токмок": 71443,
}
CITY_REGION_OVERRIDES = {
    # Привязка города к области (для фильтров UI).
    # Важно: г. Бишкек и г. Ош — города республиканского значения,
    # но для UI удобнее привязывать их к одноимённым областям.
    "Бишкек": "Чуйская",
    "Ош": "Ошская",
    "Токмок": "Чуйская",
}
REGION_POPULATION_OVERRIDES = {
    # НСК КР: "Численность постоянного населения на начало года" (2025),
    # значения переведены из тыс. человек в человек.
    "Баткенская": 594700,
    "Джалал-Абадская": 1358500,
    "Иссык-Кульская": 549800,
    "Нарынская": 314900,
    "Ошская": 1416700,
    "Таласская": 280500,
    "Чуйская": 971300,
}


def normalize(value: str | None) -> str:
    if not value:
        return ""
    normalized = (
        value.replace(" район", "")
        .replace("Район", "")
        .replace(" ", "")
        .replace(".", "")
        .replace("-", "")
        .replace("ё", "е")
        .lower()
    )
    aliases = {
        "каракульджинский": "каракулжинский",
        "атбашинский": "атбашынский",
        "айтматовский": "карабууринский",
    }
    return aliases.get(normalized, normalized)


def make_hid(value: str | None) -> str | None:
    if not value:
        return None
    normalized = normalize(value)
    return hashlib.sha1(normalized.encode("utf-8")).hexdigest()[:16]


def to_lat_lng_coords(geometry: dict) -> list:
    geometry_type = geometry.get("type")
    coordinates = geometry.get("coordinates", [])

    def ring_to_lat_lng(ring):
        return [[point[1], point[0]] for point in ring]

    if geometry_type == "Polygon":
        return [ring_to_lat_lng(ring) for ring in coordinates if ring]

    if geometry_type == "MultiPolygon":
        result = []
        for polygon in coordinates:
            for ring in polygon:
                if ring:
                    result.append(ring_to_lat_lng(ring))
        return result

    return []


def centroid_from_feature(feature: dict) -> tuple[float | None, float | None]:
    props = feature.get("properties", {})
    lat = props.get("center_lat")
    lng = props.get("center_lon")
    return lat, lng


def load_geojson(path: Path) -> dict:
    with path.open(encoding="utf-8") as file:
        return json.load(file)


def load_fallback_districts() -> dict:
    if not FALLBACK_DISTRICTS_PATH.exists():
        return {}
    try:
        with FALLBACK_DISTRICTS_PATH.open(encoding="utf-8") as file:
            data = json.load(file)
    except Exception:
        return {}

    index = {}
    if isinstance(data, list):
        for item in data:
            title = item.get("title")
            if title:
                index[normalize(title)] = item
    return index


def build_admin_indexes():
    admin1 = load_geojson(SRC_DIR / "kgz_admin1.geojson")
    admin2 = load_geojson(SRC_DIR / "kgz_admin2.geojson")
    admin3 = load_geojson(SRC_DIR / "kgz_admin3.geojson")

    oblast_index = {}
    for feature in admin1.get("features", []):
        name_ru = feature.get("properties", {}).get("adm1_name1")
        if name_ru:
            oblast_index[normalize(name_ru)] = feature

    district_index = {}
    for feature in admin2.get("features", []):
        name_ru = feature.get("properties", {}).get("adm2_name1")
        if name_ru:
            key = normalize(name_ru)
            district_index.setdefault(key, []).append(feature)

    district3_index = {}
    for feature in admin3.get("features", []):
        name_ru = feature.get("properties", {}).get("adm3_name1")
        if name_ru:
            key = normalize(name_ru)
            district3_index.setdefault(key, []).append(feature)

    return oblast_index, district_index, district3_index


def match_district_feature(item: dict, district_index: dict, district3_index: dict):
    district_key = normalize(item["район"])
    oblast_key = normalize(item["область"])

    candidates = []
    candidates.extend(district_index.get(district_key, []))
    candidates.extend(district3_index.get(district_key, []))

    if not candidates:
        return None

    for feature in candidates:
        feature_oblast = normalize(feature.get("properties", {}).get("adm1_name1"))
        if feature_oblast == oblast_key:
            return feature

    return candidates[0]


def make_districts(district_index: dict, district3_index: dict, fallback_districts: dict):
    bishkek_items = []
    for fallback in fallback_districts.values():
        raw_title = fallback.get("title", "")
        district_title = raw_title.replace(" район", "").strip()
        bishkek_items.append(
            {
                "title": district_title,
                "population": fallback.get("population"),
                "lat": fallback.get("lat"),
                "lng": fallback.get("lng"),
                "coords": fallback.get("coords", []),
                "hid": make_hid(district_title),
                "city_hid": make_hid("Бишкек"),
                "region_hid": make_hid("г. Бишкек"),
            }
        )

    result = bishkek_items
    for item in kyrgyzstan_data:
        district_name = item["район"]
        city_name = item["город"]
        region_name = item["область"]
        feature = match_district_feature(item, district_index, district3_index)
        lat, lng = (None, None)
        coords = []
        if feature:
            lat, lng = centroid_from_feature(feature)
            coords = to_lat_lng_coords(feature.get("geometry", {}))
        elif district_name in BISHKEK_DISTRICTS:
            fallback = fallback_districts.get(normalize(district_name))
            if fallback:
                lat = fallback.get("lat")
                lng = fallback.get("lng")
                coords = fallback.get("coords", [])

        result.append(
            {
                "title": district_name,
                "population": item["население"],
                "lat": lat,
                "lng": lng,
                "coords": coords,
                "hid": make_hid(district_name),
                "city_hid": make_hid(city_name),
                "region_hid": make_hid(region_name),
            }
        )

    return result


def make_regions(oblast_index: dict):
    population_by_oblast = {}
    for item in kyrgyzstan_data:
        population_by_oblast[item["область"]] = population_by_oblast.get(item["область"], 0) + item["население"]

    result = []
    for oblast_name, population in population_by_oblast.items():
        if normalize(oblast_name).startswith("г"):
            continue
        population = REGION_POPULATION_OVERRIDES.get(oblast_name, population)
        feature = oblast_index.get(normalize(oblast_name))
        lat, lng = (None, None)
        coords = []
        if feature:
            lat, lng = centroid_from_feature(feature)
            coords = to_lat_lng_coords(feature.get("geometry", {}))

        result.append(
            {
                "title": oblast_name,
                "population": population,
                "lat": lat,
                "lng": lng,
                "coords": coords,
                "hid": make_hid(oblast_name),
            }
        )
    return result


def make_cities(oblast_index: dict, district_index: dict, district3_index: dict):
    population_by_city = {}
    for item in kyrgyzstan_data:
        if item["город"]:
            population_by_city[item["город"]] = population_by_city.get(item["город"], 0) + item["население"]

    for city_name, population in CITY_POPULATION_OVERRIDES.items():
        if city_name not in population_by_city:
            population_by_city[city_name] = population

    result = []
    for city_name, population in population_by_city.items():
        feature = oblast_index.get(normalize(f"г.{city_name}")) or oblast_index.get(normalize(city_name))
        region_name = CITY_REGION_OVERRIDES.get(city_name)
        lat, lng = (None, None)
        coords = []
        if not feature:
            city_key = normalize(city_name)
            city_key_g = normalize(f"г.{city_name}")
            candidates = []
            candidates.extend(district_index.get(city_key_g, []))
            candidates.extend(district_index.get(city_key, []))
            candidates.extend(district3_index.get(city_key_g, []))
            candidates.extend(district3_index.get(city_key, []))
            if candidates and region_name:
                region_key = normalize(region_name)
                for cand in candidates:
                    cand_region = normalize(cand.get("properties", {}).get("adm1_name1"))
                    if cand_region == region_key:
                        feature = cand
                        break
            if not feature and candidates:
                feature = candidates[0]

        if feature:
            lat, lng = centroid_from_feature(feature)
            coords = to_lat_lng_coords(feature.get("geometry", {}))

        result.append(
            {
                "title": city_name,
                "population": population,
                "lat": lat,
                "lng": lng,
                "coords": coords,
                "hid": make_hid(city_name),
                "region_hid": make_hid(region_name),
            }
        )
    return result


def main():
    oblast_index, district_index, district3_index = build_admin_indexes()
    fallback_districts = load_fallback_districts()

    districts = make_districts(district_index, district3_index, fallback_districts)
    cities = make_cities(oblast_index, district_index, district3_index)
    regions = make_regions(oblast_index)

    DATA_DIR.mkdir(parents=True, exist_ok=True)
    (DATA_DIR / "districts.json").write_text(json.dumps(districts, ensure_ascii=False, indent=2), encoding="utf-8")
    (DATA_DIR / "cities.json").write_text(json.dumps(cities, ensure_ascii=False, indent=2), encoding="utf-8")
    (DATA_DIR / "regions.json").write_text(json.dumps(regions, ensure_ascii=False, indent=2), encoding="utf-8")

    print(f"districts: {len(districts)}")
    print(f"cities: {len(cities)}")
    print(f"regions: {len(regions)}")
    print(f"saved: {DATA_DIR / 'districts.json'}")
    print(f"saved: {DATA_DIR / 'cities.json'}")
    print(f"saved: {DATA_DIR / 'regions.json'}")


if __name__ == "__main__":
    main()
