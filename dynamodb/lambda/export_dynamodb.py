import os
import json
import boto3
from datetime import datetime
from zoneinfo import ZoneInfo


def handler(event, context):
    table_name = os.environ["TABLE_NAME"]
    bucket_name = os.environ["BUCKET_NAME"]
    prefix_base = os.environ.get("EXPORT_PREFIX_BASE", "exports/core")
    export_format = os.environ.get("EXPORT_FORMAT", "AMAZON_ION")

    dynamodb = boto3.client("dynamodb")
    table = dynamodb.describe_table(TableName=table_name)
    table_arn = table["Table"]["TableArn"]

    sao_paulo = ZoneInfo("America/Sao_Paulo")
    date_str = datetime.now(tz=sao_paulo).strftime("%Y-%m-%d")
    export_prefix = f"{prefix_base}/{date_str}/"

    response = dynamodb.export_table_to_point_in_time(
        TableArn=table_arn,
        S3Bucket=bucket_name,
        S3Prefix=export_prefix,
        ExportFormat=export_format
    )

    export_arn = response.get("ExportDescription", {}).get("ExportArn", "")
    print(json.dumps({
        "message": "Export triggered",
        "table": table_name,
        "bucket": bucket_name,
        "prefix": export_prefix,
        "export_arn": export_arn
    }))

    return {
        "statusCode": 200,
        "body": json.dumps({"export_arn": export_arn})
    }
