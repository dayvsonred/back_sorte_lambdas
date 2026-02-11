import argparse
import json
from datetime import datetime
from pathlib import Path
import re
from typing import Dict, List

import boto3
from botocore.exceptions import ClientError

BASE_DIR = Path(__file__).resolve().parent
DATE_RE = re.compile(r"^\d{4}-\d{2}-\d{2}$")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Download DynamoDB export files from S3 to local disk."
    )
    parser.add_argument("--region", default="us-east-1", help="AWS region")
    parser.add_argument(
        "--bucket",
        default="bd-thepuregrace-v1-dinamodb-core",
        help="S3 bucket with exports",
    )
    parser.add_argument(
        "--prefix-base",
        default="exports/core",
        help="Base prefix used by DynamoDB exports",
    )
    parser.add_argument(
        "--date",
        default=None,
        help="Single export date folder (YYYY-MM-DD). If omitted, use --from-last behavior.",
    )
    parser.add_argument(
        "--from-last",
        action="store_true",
        help="Download from first date after the latest local date up to latest date available on S3.",
    )
    parser.add_argument(
        "--all-dates",
        action="store_true",
        help="Download all date folders available on S3 under prefix-base.",
    )
    parser.add_argument(
        "--out-dir",
        default=str(BASE_DIR / "data" / "raw"),
        help="Local output directory",
    )
    return parser.parse_args()


def build_local_path(out_dir: Path, key: str) -> Path:
    return out_dir / Path(*key.split("/"))


def list_s3_dates(s3: boto3.client, bucket: str, prefix_base: str) -> List[str]:
    prefix_root = f"{prefix_base}/"
    paginator = s3.get_paginator("list_objects_v2")
    dates: List[str] = []

    for page in paginator.paginate(Bucket=bucket, Prefix=prefix_root, Delimiter="/"):
        for cp in page.get("CommonPrefixes", []):
            pfx = cp.get("Prefix", "")
            value = pfx.removeprefix(prefix_root).strip("/")
            if DATE_RE.match(value):
                dates.append(value)

    return sorted(set(dates))


def list_local_dates(out_dir: Path, prefix_base: str) -> List[str]:
    base_dir = out_dir / Path(*prefix_base.split("/"))
    if not base_dir.exists():
        return []

    dates = [p.name for p in base_dir.iterdir() if p.is_dir() and DATE_RE.match(p.name)]
    return sorted(dates)


def resolve_dates(args: argparse.Namespace, s3_dates: List[str], local_dates: List[str]) -> List[str]:
    if args.date:
        if not DATE_RE.match(args.date):
            raise SystemExit("--date must be in format YYYY-MM-DD")
        return [args.date]

    if args.all_dates:
        return s3_dates

    # Default behavior if user did not pass any explicit selector.
    use_from_last = args.from_last or (not args.date and not args.all_dates)
    if use_from_last:
        if not s3_dates:
            return []
        if not local_dates:
            # First run: avoid huge backfill by default.
            latest = s3_dates[-1]
            print(
                f"[info] no local history found. downloading latest available date only: {latest}"
            )
            return [latest]
        last_local = local_dates[-1]
        return [d for d in s3_dates if d > last_local]

    return []


def download_prefix(
    s3: boto3.client, bucket: str, prefix: str, out_dir: Path
) -> Dict[str, int]:
    paginator = s3.get_paginator("list_objects_v2")
    downloaded = 0
    skipped = 0
    total_bytes = 0
    found_any = False

    for page in paginator.paginate(Bucket=bucket, Prefix=prefix):
        for obj in page.get("Contents", []):
            found_any = True
            key = obj["Key"]
            size = int(obj["Size"])

            if key.endswith("/"):
                continue

            local_path = build_local_path(out_dir, key)
            local_path.parent.mkdir(parents=True, exist_ok=True)

            if local_path.exists() and local_path.stat().st_size == size:
                skipped += 1
                continue

            try:
                s3.download_file(bucket, key, str(local_path))
                downloaded += 1
                total_bytes += size
                print(f"[download] {key}")
            except ClientError as exc:
                print(f"[error] failed to download {key}: {exc}")

    return {
        "found_any": int(found_any),
        "downloaded": downloaded,
        "skipped": skipped,
        "downloaded_bytes": total_bytes,
    }


def main() -> None:
    args = parse_args()
    s3 = boto3.client("s3", region_name=args.region)

    out_dir = Path(args.out_dir).resolve()
    out_dir.mkdir(parents=True, exist_ok=True)

    s3_dates = list_s3_dates(s3, args.bucket, args.prefix_base)
    local_dates = list_local_dates(out_dir, args.prefix_base)
    target_dates = resolve_dates(args, s3_dates, local_dates)

    if not target_dates:
        print("[info] no new dates to download.")
        return

    total_downloaded = 0
    total_skipped = 0
    total_bytes = 0
    downloaded_dates: List[str] = []

    for date_str in target_dates:
        prefix = f"{args.prefix_base}/{date_str}/"
        print(f"[info] syncing s3://{args.bucket}/{prefix}")
        result = download_prefix(s3, args.bucket, prefix, out_dir)
        if result["found_any"] == 0:
            print(f"[warn] no objects found under s3://{args.bucket}/{prefix}")
            continue
        downloaded_dates.append(date_str)
        total_downloaded += result["downloaded"]
        total_skipped += result["skipped"]
        total_bytes += result["downloaded_bytes"]

    summary = {
        "bucket": args.bucket,
        "prefix_base": args.prefix_base,
        "dates_requested": target_dates,
        "dates_downloaded": downloaded_dates,
        "out_dir": str(out_dir),
        "downloaded_files": total_downloaded,
        "skipped_files": total_skipped,
        "downloaded_bytes": total_bytes,
        "finished_at": datetime.utcnow().isoformat() + "Z",
    }

    meta_path = out_dir / "last_sync.json"
    meta_path.write_text(json.dumps(summary, indent=2), encoding="utf-8")

    print(json.dumps(summary, indent=2))


if __name__ == "__main__":
    main()
