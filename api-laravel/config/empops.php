<?php

return [
    'enable_signups' => (bool) env('ENABLE_SIGNUPS', true),
    'demo_mode' => (bool) env('DEMO_MODE', false),
    'enable_paid_plan' => (bool) env('ENABLE_PAID_PLAN', false),
];
