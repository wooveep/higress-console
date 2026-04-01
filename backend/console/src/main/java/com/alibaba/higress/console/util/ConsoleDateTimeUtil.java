package com.alibaba.higress.console.util;

import java.sql.Timestamp;
import java.time.Instant;
import java.time.LocalDateTime;
import java.time.OffsetDateTime;
import java.time.ZoneId;
import java.time.ZonedDateTime;
import java.time.format.DateTimeFormatter;
import java.time.format.DateTimeParseException;

import org.apache.commons.lang3.StringUtils;

import com.alibaba.higress.sdk.exception.ValidationException;

public final class ConsoleDateTimeUtil {

    public static final String APP_TIME_ZONE = "UTC";
    public static final ZoneId APP_ZONE_ID = ZoneId.of(APP_TIME_ZONE);

    private static final String DEFAULT_LOCAL_DATE_TIME_FORMAT_DESCRIPTION =
        "RFC3339 or yyyy-MM-dd'T'HH:mm[:ss] in UTC";
    private static final DateTimeFormatter LOCAL_DATE_TIME_MINUTE_FORMATTER =
        DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm");

    private ConsoleDateTimeUtil() {
    }

    public static LocalDateTime now() {
        return LocalDateTime.now(APP_ZONE_ID);
    }

    public static Timestamp nowTimestamp() {
        return Timestamp.from(Instant.now());
    }

    public static ZonedDateTime atAppZone(long epochMilli) {
        return Instant.ofEpochMilli(epochMilli).atZone(APP_ZONE_ID);
    }

    public static String formatTimestamp(Timestamp timestamp) {
        return timestamp == null ? null : timestamp.toInstant().toString();
    }

    public static String formatLocalDateTime(LocalDateTime value) {
        return value == null ? null : value.atZone(APP_ZONE_ID).toInstant().toString();
    }

    public static Timestamp toTimestamp(LocalDateTime value) {
        return value == null ? null : Timestamp.from(value.atZone(APP_ZONE_ID).toInstant());
    }

    public static LocalDateTime toLocalDateTime(Timestamp timestamp) {
        return timestamp == null ? null : timestamp.toInstant().atZone(APP_ZONE_ID).toLocalDateTime();
    }

    public static Timestamp parseTimestamp(String value, String fieldName) {
        return parseTimestamp(value, fieldName, DEFAULT_LOCAL_DATE_TIME_FORMAT_DESCRIPTION);
    }

    public static Timestamp parseTimestamp(String value, String fieldName, String formatDescription) {
        String normalized = StringUtils.trimToEmpty(value);
        if (normalized.isEmpty()) {
            return null;
        }
        try {
            return Timestamp.from(Instant.parse(normalized));
        } catch (DateTimeParseException ignored) {
        }
        try {
            return Timestamp.from(OffsetDateTime.parse(normalized).toInstant());
        } catch (DateTimeParseException ignored) {
        }
        try {
            return Timestamp.from(LocalDateTime.parse(normalized, DateTimeFormatter.ISO_LOCAL_DATE_TIME)
                .atZone(APP_ZONE_ID).toInstant());
        } catch (DateTimeParseException ignored) {
        }
        try {
            return Timestamp.from(LocalDateTime.parse(normalized, LOCAL_DATE_TIME_MINUTE_FORMATTER)
                .atZone(APP_ZONE_ID).toInstant());
        } catch (DateTimeParseException ex) {
            throw new ValidationException(fieldName + " must be " + formatDescription + ".");
        }
    }
}
