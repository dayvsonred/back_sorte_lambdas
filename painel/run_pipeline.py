import argparse
import subprocess
import sys
from pathlib import Path


BASE_DIR = Path(__file__).resolve().parent


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Run full local pipeline: sync S3 -> build dataset -> open Streamlit."
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
        help="Base export prefix in S3",
    )
    parser.add_argument(
        "--date",
        default=None,
        help="Specific date (YYYY-MM-DD). If omitted, sync from last local date.",
    )
    parser.add_argument(
        "--all-dates",
        action="store_true",
        help="Sync all dates from S3 instead of incremental mode.",
    )
    parser.add_argument(
        "--from-last",
        action="store_true",
        help="Sync from first date after latest local date up to latest on S3.",
    )
    parser.add_argument(
        "--no-open",
        action="store_true",
        help="Do not open Streamlit after building dataset.",
    )
    return parser.parse_args()


def run_cmd(cmd: list[str], step_label: str) -> None:
    print(step_label)
    print(f"[cmd] {' '.join(cmd)}")
    result = subprocess.run(cmd, cwd=str(BASE_DIR))
    if result.returncode != 0:
        raise SystemExit(result.returncode)


def main() -> None:
    args = parse_args()

    sync_cmd = [
        sys.executable,
        str(BASE_DIR / "sync_s3_export.py"),
        "--region",
        args.region,
        "--bucket",
        args.bucket,
        "--prefix-base",
        args.prefix_base,
    ]
    if args.date:
        sync_cmd.extend(["--date", args.date])
    elif args.all_dates:
        sync_cmd.append("--all-dates")
    else:
        sync_cmd.append("--from-last")

    run_cmd(sync_cmd, "[etapa 1/3] Baixando dados do S3")

    build_cmd = [sys.executable, str(BASE_DIR / "build_dataset.py")]
    run_cmd(build_cmd, "[etapa 2/3] Gerando Parquet e DuckDB")

    if args.no_open:
        print("[etapa 3/3] Streamlit nao iniciado (--no-open).")
        print("[ok] Pipeline finalizado.")
        return

    streamlit_cmd = [sys.executable, "-m", "streamlit", "run", str(BASE_DIR / "app.py")]
    run_cmd(streamlit_cmd, "[etapa 3/3] Abrindo painel Streamlit")


if __name__ == "__main__":
    main()
