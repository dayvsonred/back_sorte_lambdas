import argparse
import gzip
import io
import json
from datetime import datetime
from decimal import Decimal
from pathlib import Path
from typing import Any, Dict, Iterable, List, Optional

import duckdb
import pandas as pd

BASE_DIR = Path(__file__).resolve().parent


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Convert DynamoDB export files (DYNAMODB_JSON) to Parquet and DuckDB."
    )
    parser.add_argument(
        "--raw-dir",
        default=str(BASE_DIR / "data" / "raw"),
        help="Directory with downloaded export files",
    )
    parser.add_argument(
        "--out-dir",
        default=str(BASE_DIR / "data" / "curated"),
        help="Output directory for Parquet files",
    )
    parser.add_argument(
        "--db-path",
        default=str(BASE_DIR / "data" / "dashboard.duckdb"),
        help="DuckDB file path",
    )
    return parser.parse_args()


def ddb_value_to_python(value: Dict[str, Any]) -> Any:
    if "S" in value:
        return value["S"]
    if "N" in value:
        return Decimal(value["N"])
    if "BOOL" in value:
        return bool(value["BOOL"])
    if "NULL" in value:
        return None
    if "M" in value:
        return {k: ddb_value_to_python(v) for k, v in value["M"].items()}
    if "L" in value:
        return [ddb_value_to_python(v) for v in value["L"]]
    if "SS" in value:
        return value["SS"]
    if "NS" in value:
        return [Decimal(v) for v in value["NS"]]
    return None


def parse_item(item: Dict[str, Any]) -> Dict[str, Any]:
    return {k: ddb_value_to_python(v) for k, v in item.items()}


def parse_decimal(value: Any) -> Optional[float]:
    if value is None:
        return None
    try:
        return float(value)
    except Exception:
        return None


def parse_datetime(value: Any) -> Optional[datetime]:
    if not value or not isinstance(value, str):
        return None
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        return None


def iter_export_items(raw_dir: Path) -> Iterable[Dict[str, Any]]:
    pattern = "**/AWSDynamoDB/*/data/*.json.gz"
    for gz_file in raw_dir.glob(pattern):
        with gz_file.open("rb") as fh:
            content = fh.read()
        with gzip.GzipFile(fileobj=io.BytesIO(content), mode="rb") as gz:
            for line in gz:
                row = json.loads(line.decode("utf-8"))
                if "Item" not in row:
                    continue
                yield parse_item(row["Item"])


def build_frames(items: Iterable[Dict[str, Any]]) -> Dict[str, pd.DataFrame]:
    users: List[Dict[str, Any]] = []
    donations: List[Dict[str, Any]] = []
    donation_links: List[Dict[str, Any]] = []
    accesses: List[Dict[str, Any]] = []
    payments: List[Dict[str, Any]] = []

    for item in items:
        pk = str(item.get("PK", ""))
        sk = str(item.get("SK", ""))

        if pk.startswith("USER#") and sk == "PROFILE":
            users.append(
                {
                    "user_id": pk.removeprefix("USER#"),
                    "email": item.get("email"),
                    "name": item.get("name"),
                    "active": item.get("active"),
                    "created_at": parse_datetime(item.get("date_create")),
                }
            )
            continue

        if pk.startswith("DONATION#") and sk == "PROFILE":
            donations.append(
                {
                    "donation_id": pk.removeprefix("DONATION#"),
                    "user_id": item.get("id_user"),
                    "name": item.get("name"),
                    "amount_declared": parse_decimal(item.get("valor")),
                    "closed": item.get("closed"),
                    "active": item.get("active"),
                    "created_at": parse_datetime(item.get("date_create")),
                }
            )
            continue

        if pk.startswith("LINK#") and sk.startswith("DONATION#"):
            donation_links.append(
                {
                    "donation_id": sk.removeprefix("DONATION#"),
                    "link_name": pk.removeprefix("LINK#"),
                    "link_url": f"https://www.thepuregrace.com/{pk.removeprefix('LINK#')}",
                }
            )
            continue

        if pk.startswith("DONATION#") and sk.startswith("VIS#"):
            accesses.append(
                {
                    "donation_id": pk.removeprefix("DONATION#"),
                    "event_key": sk,
                    "user_id": item.get("id_user"),
                    "acesse_donation": bool(item.get("acesse_donation", False)),
                    "create_pix": bool(item.get("create_pix", False)),
                    "create_cartao": bool(item.get("create_cartao", False)),
                    "create_paypal": bool(item.get("create_paypal", False)),
                    "create_google": bool(item.get("create_google", False)),
                    "create_pag1": bool(item.get("create_pag1", False)),
                    "create_pag2": bool(item.get("create_pag2", False)),
                    "create_pag3": bool(item.get("create_pag3", False)),
                    "created_at": parse_datetime(item.get("date_create")),
                }
            )
            continue

        if pk.startswith("TX#") and sk == "STATUS":
            payments.append(
                {
                    "txid": pk.removeprefix("TX#"),
                    "donation_id": item.get("id_doacao"),
                    "status": item.get("status"),
                    "finalizado": bool(item.get("finalizado", False)),
                    "tipo_pagamento": item.get("tipo_pagamento"),
                    "amount": parse_decimal(item.get("valor")),
                    "paid_at": parse_datetime(item.get("data_pago")),
                    "created_at": parse_datetime(item.get("date_create")),
                }
            )

    users_cols = ["user_id", "email", "name", "active", "created_at"]
    donations_cols = [
        "donation_id",
        "user_id",
        "name",
        "amount_declared",
        "closed",
        "active",
        "created_at",
    ]
    accesses_cols = [
        "donation_id",
        "event_key",
        "user_id",
        "acesse_donation",
        "create_pix",
        "create_cartao",
        "create_paypal",
        "create_google",
        "create_pag1",
        "create_pag2",
        "create_pag3",
        "created_at",
    ]
    donation_links_cols = ["donation_id", "link_name", "link_url"]
    payments_cols = [
        "txid",
        "donation_id",
        "status",
        "finalizado",
        "tipo_pagamento",
        "amount",
        "paid_at",
        "created_at",
    ]

    return {
        "users": pd.DataFrame(users, columns=users_cols),
        "donations": pd.DataFrame(donations, columns=donations_cols),
        "donation_links": pd.DataFrame(donation_links, columns=donation_links_cols),
        "accesses": pd.DataFrame(accesses, columns=accesses_cols),
        "payments": pd.DataFrame(payments, columns=payments_cols),
    }


def save_parquet(frames: Dict[str, pd.DataFrame], out_dir: Path) -> None:
    out_dir.mkdir(parents=True, exist_ok=True)
    for name, df in frames.items():
        path = out_dir / f"{name}.parquet"
        df.to_parquet(path, index=False)
        print(f"[write] {path} ({len(df)} rows)")


def build_duckdb(frames: Dict[str, pd.DataFrame], db_path: Path) -> None:
    db_path.parent.mkdir(parents=True, exist_ok=True)
    with duckdb.connect(str(db_path)) as con:
        for name, df in frames.items():
            con.register(f"{name}_df", df)
            con.execute(f"CREATE OR REPLACE TABLE {name} AS SELECT * FROM {name}_df")
            con.unregister(f"{name}_df")

        con.execute(
            """
            CREATE OR REPLACE VIEW daily_users AS
            SELECT CAST(created_at AS DATE) AS day, COUNT(*) AS users_created
            FROM users
            WHERE created_at IS NOT NULL
            GROUP BY 1
            ORDER BY 1
            """
        )
        con.execute(
            """
            CREATE OR REPLACE VIEW daily_donations AS
            SELECT CAST(created_at AS DATE) AS day, COUNT(*) AS donations_created
            FROM donations
            WHERE created_at IS NOT NULL
            GROUP BY 1
            ORDER BY 1
            """
        )
        con.execute(
            """
            CREATE OR REPLACE VIEW donation_details AS
            SELECT
              CAST(d.created_at AS DATE) AS day,
              d.created_at,
              d.donation_id,
              d.user_id,
              d.name AS donation_name,
              d.amount_declared,
              d.active,
              d.closed,
              l.link_name,
              l.link_url
            FROM donations d
            LEFT JOIN donation_links l ON l.donation_id = d.donation_id
            ORDER BY d.created_at DESC
            """
        )
        con.execute(
            """
            CREATE OR REPLACE VIEW daily_accesses AS
            SELECT
              CAST(created_at AS DATE) AS day,
              COUNT(*) AS access_events,
              SUM(CASE WHEN acesse_donation THEN 1 ELSE 0 END) AS donation_page_accesses,
              SUM(CASE WHEN create_pag1 OR create_pag2 OR create_pag3 THEN 1 ELSE 0 END) AS page_click_events
            FROM accesses
            WHERE created_at IS NOT NULL
            GROUP BY 1
            ORDER BY 1
            """
        )
        con.execute(
            """
            CREATE OR REPLACE VIEW daily_access_breakdown AS
            SELECT
              CAST(created_at AS DATE) AS day,
              COUNT(*) AS total_events,
              COUNT(DISTINCT NULLIF(user_id, '')) AS unique_users,
              SUM(CASE WHEN acesse_donation THEN 1 ELSE 0 END) AS acesse_donation,
              SUM(CASE WHEN create_pag1 THEN 1 ELSE 0 END) AS create_pag1,
              SUM(CASE WHEN create_pag2 THEN 1 ELSE 0 END) AS create_pag2,
              SUM(CASE WHEN create_pag3 THEN 1 ELSE 0 END) AS create_pag3,
              SUM(CASE WHEN create_pix THEN 1 ELSE 0 END) AS create_pix,
              SUM(CASE WHEN create_cartao THEN 1 ELSE 0 END) AS create_cartao,
              SUM(CASE WHEN create_paypal THEN 1 ELSE 0 END) AS create_paypal,
              SUM(CASE WHEN create_google THEN 1 ELSE 0 END) AS create_google
            FROM accesses
            WHERE created_at IS NOT NULL
            GROUP BY 1
            ORDER BY 1
            """
        )
        con.execute(
            """
            CREATE OR REPLACE VIEW access_by_donation AS
            SELECT
              donation_id,
              COUNT(*) AS total_events,
              SUM(CASE WHEN acesse_donation THEN 1 ELSE 0 END) AS donation_page_accesses,
              SUM(CASE WHEN create_pag1 OR create_pag2 OR create_pag3 THEN 1 ELSE 0 END) AS page_click_events
            FROM accesses
            GROUP BY 1
            ORDER BY total_events DESC
            """
        )
        con.execute(
            """
            CREATE OR REPLACE VIEW daily_payments AS
            SELECT
              CAST(COALESCE(paid_at, created_at) AS DATE) AS day,
              COUNT(*) FILTER (WHERE finalizado) AS finalized_payments,
              SUM(CASE WHEN finalizado THEN COALESCE(amount, 0) ELSE 0 END) AS finalized_amount
            FROM payments
            WHERE COALESCE(paid_at, created_at) IS NOT NULL
            GROUP BY 1
            ORDER BY 1
            """
        )

    print(f"[write] {db_path}")


def main() -> None:
    args = parse_args()
    raw_dir = Path(args.raw_dir).resolve()
    out_dir = Path(args.out_dir).resolve()
    db_path = Path(args.db_path).resolve()

    if not raw_dir.exists():
        raise SystemExit(f"raw directory not found: {raw_dir}")

    frames = build_frames(iter_export_items(raw_dir))
    save_parquet(frames, out_dir)
    build_duckdb(frames, db_path)


if __name__ == "__main__":
    main()
