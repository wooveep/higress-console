package com.alibaba.higress.console.util;

import java.sql.Timestamp;
import java.time.LocalDateTime;

public final class AiSensitiveDateTimeUtil {

    private AiSensitiveDateTimeUtil() {
    }

    public static String formatTimestamp(Timestamp timestamp) {
        return ConsoleDateTimeUtil.formatTimestamp(timestamp);
    }

    public static String formatLocalDateTime(LocalDateTime value) {
        return ConsoleDateTimeUtil.formatLocalDateTime(value);
    }

    public static Timestamp parseTimestamp(String value, String fieldName) {
        return ConsoleDateTimeUtil.parseTimestamp(value, fieldName);
    }
}
