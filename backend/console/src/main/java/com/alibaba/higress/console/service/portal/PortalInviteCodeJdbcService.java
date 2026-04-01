package com.alibaba.higress.console.service.portal;

import java.security.SecureRandom;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Timestamp;
import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.List;
import java.util.Locale;

import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import com.alibaba.higress.console.model.portal.PortalInviteCodePageQuery;
import com.alibaba.higress.console.model.portal.PortalInviteCodeRecord;
import com.alibaba.higress.console.util.ConsoleDateTimeUtil;
import com.alibaba.higress.sdk.exception.BusinessException;
import com.alibaba.higress.sdk.exception.ValidationException;
import com.alibaba.higress.sdk.model.PaginatedResult;

import lombok.extern.slf4j.Slf4j;

@Slf4j
@Service
public class PortalInviteCodeJdbcService {

    private static final String STATUS_ACTIVE = "active";
    private static final String STATUS_DISABLED = "disabled";
    private static final int DEFAULT_EXPIRE_DAYS = 7;
    private static final int MAX_EXPIRE_DAYS = 365;
    private static final int INVITE_CODE_LENGTH = 16;
    private static final int MAX_GENERATE_ATTEMPTS = 10;
    private static final char[] INVITE_CODE_CHARS =
        "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789".toCharArray();

    @Value("${higress.portal.db.url:}")
    private String dbUrl;

    @Value("${higress.portal.db.username:}")
    private String dbUsername;

    @Value("${higress.portal.db.password:}")
    private String dbPassword;

    private final SecureRandom secureRandom = new SecureRandom();

    public boolean enabled() {
        return StringUtils.isNotBlank(dbUrl);
    }

    public PortalInviteCodeRecord createInviteCode(Integer expiresInDays) {
        if (!enabled()) {
            throw new IllegalStateException("Portal database is unavailable.");
        }
        int validDays = normalizeExpireDays(expiresInDays);
        LocalDateTime expiresAt = ConsoleDateTimeUtil.now().plusDays(validDays);
        String insertSql = "INSERT INTO portal_invite_code (invite_code, status, expires_at) VALUES (?, ?, ?)";

        for (int i = 0; i < MAX_GENERATE_ATTEMPTS; i++) {
            String inviteCode = generateInviteCode();
            try (Connection connection = openConnection();
                PreparedStatement statement = connection.prepareStatement(insertSql)) {
                statement.setString(1, inviteCode);
                statement.setString(2, STATUS_ACTIVE);
                statement.setTimestamp(3, ConsoleDateTimeUtil.toTimestamp(expiresAt));
                statement.executeUpdate();
                return queryByInviteCode(inviteCode);
            } catch (SQLException ex) {
                if (isDuplicateKey(ex)) {
                    continue;
                }
                log.warn("Failed to create invite code.", ex);
                throw new BusinessException("Failed to create invite code.", ex);
            }
        }
        throw new BusinessException("Failed to create invite code after retries.");
    }

    public PaginatedResult<PortalInviteCodeRecord> list(PortalInviteCodePageQuery query) {
        if (!enabled()) {
            throw new IllegalStateException("Portal database is unavailable.");
        }
        String sql =
            "SELECT invite_code, status, expires_at, used_by_consumer, used_at, created_at "
                + "FROM portal_invite_code ORDER BY created_at DESC";
        List<PortalInviteCodeRecord> records = new ArrayList<>();
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql);
            ResultSet rs = statement.executeQuery()) {
            while (rs.next()) {
                PortalInviteCodeRecord record = mapRecord(rs);
                if (query == null || StringUtils.isBlank(query.getStatus())
                    || StringUtils.equalsIgnoreCase(query.getStatus(), record.getStatus())) {
                    records.add(record);
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to list invite codes.", ex);
            throw new BusinessException("Failed to list invite codes.", ex);
        }
        return PaginatedResult.createFromFullList(records, query);
    }

    public PortalInviteCodeRecord disableInviteCode(String inviteCode) {
        return updateInviteCodeStatus(inviteCode, STATUS_DISABLED);
    }

    public PortalInviteCodeRecord updateInviteCodeStatus(String inviteCode, String status) {
        if (!enabled()) {
            throw new IllegalStateException("Portal database is unavailable.");
        }
        if (StringUtils.isBlank(inviteCode)) {
            throw new ValidationException("inviteCode cannot be blank.");
        }
        String normalizedStatus = StringUtils.lowerCase(StringUtils.trimToEmpty(status));
        if (!STATUS_ACTIVE.equals(normalizedStatus) && !STATUS_DISABLED.equals(normalizedStatus)) {
            throw new ValidationException("status must be 'active' or 'disabled'.");
        }
        String normalizedCode = inviteCode.trim();
        String sql = "UPDATE portal_invite_code SET status = ? WHERE invite_code = ?";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, normalizedStatus);
            statement.setString(2, normalizedCode);
            int affectedRows = statement.executeUpdate();
            if (affectedRows <= 0) {
                throw new ValidationException("Invite code not found: " + normalizedCode);
            }
        } catch (SQLException ex) {
            log.warn("Failed to update invite code {} status to {}.", normalizedCode, normalizedStatus, ex);
            throw new BusinessException("Failed to update invite code status.", ex);
        }

        PortalInviteCodeRecord record = queryByInviteCode(normalizedCode);
        if (record == null) {
            throw new ValidationException("Invite code not found: " + normalizedCode);
        }
        return record;
    }

    public PortalInviteCodeRecord queryByInviteCode(String inviteCode) {
        if (!enabled() || StringUtils.isBlank(inviteCode)) {
            return null;
        }
        String sql =
            "SELECT invite_code, status, expires_at, used_by_consumer, used_at, created_at "
                + "FROM portal_invite_code WHERE invite_code = ?";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, inviteCode);
            try (ResultSet rs = statement.executeQuery()) {
                if (rs.next()) {
                    return mapRecord(rs);
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to query invite code {}.", inviteCode, ex);
        }
        return null;
    }

    private int normalizeExpireDays(Integer expiresInDays) {
        int validDays = expiresInDays == null ? DEFAULT_EXPIRE_DAYS : expiresInDays;
        if (validDays <= 0 || validDays > MAX_EXPIRE_DAYS) {
            throw new ValidationException("expiresInDays must be between 1 and " + MAX_EXPIRE_DAYS + ".");
        }
        return validDays;
    }

    private String generateInviteCode() {
        char[] data = new char[INVITE_CODE_LENGTH];
        for (int i = 0; i < data.length; i++) {
            data[i] = INVITE_CODE_CHARS[secureRandom.nextInt(INVITE_CODE_CHARS.length)];
        }
        return new String(data);
    }

    private boolean isDuplicateKey(SQLException ex) {
        String sqlState = ex.getSQLState();
        if ("23000".equals(sqlState) || "23505".equals(sqlState)) {
            return true;
        }
        String message = StringUtils.defaultString(ex.getMessage()).toLowerCase(Locale.ROOT);
        return message.contains("duplicate") || message.contains("unique");
    }

    private Connection openConnection() throws SQLException {
        if (StringUtils.isBlank(dbUsername)) {
            return DriverManager.getConnection(dbUrl);
        }
        return DriverManager.getConnection(dbUrl, dbUsername, dbPassword);
    }

    private PortalInviteCodeRecord mapRecord(ResultSet rs) throws SQLException {
        return PortalInviteCodeRecord.builder().inviteCode(rs.getString("invite_code")).status(rs.getString("status"))
            .expiresAt(toLocalDateTime(rs.getTimestamp("expires_at")))
            .usedByConsumer(rs.getString("used_by_consumer")).usedAt(toLocalDateTime(rs.getTimestamp("used_at")))
            .createdAt(toLocalDateTime(rs.getTimestamp("created_at"))).build();
    }

    private LocalDateTime toLocalDateTime(Timestamp timestamp) {
        if (timestamp == null) {
            return null;
        }
        return ConsoleDateTimeUtil.toLocalDateTime(timestamp);
    }
}
