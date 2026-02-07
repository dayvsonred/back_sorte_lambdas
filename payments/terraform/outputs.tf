output "endpoint_donations" {
  value = "${local.base_url}/payments/donations"
}

output "endpoint_intents" {
  value = "${local.base_url}/payments/intents"
}

output "event_bus_name" {
  value = aws_cloudwatch_event_bus.stripe.name
}

output "eventbridge_log_group" {
  value = aws_cloudwatch_log_group.stripe_events.name
}
