package com.alibaba.higress.console.service.portal;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;

import java.time.Instant;
import java.time.LocalDateTime;
import java.util.List;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.test.util.ReflectionTestUtils;

import com.alibaba.higress.console.model.aisensitive.AiSensitiveBlockAudit;
import com.alibaba.higress.console.model.aisensitive.AiSensitiveBlockAuditEvent;
import com.alibaba.higress.console.model.aisensitive.AiSensitiveSystemConfig;
import com.alibaba.higress.console.util.ConsoleDateTimeUtil;

class AiSensitiveWordJdbcServiceTest {

    private AiSensitiveWordJdbcService service;

    @BeforeEach
    void setUp() {
        service = new AiSensitiveWordJdbcService();
        ReflectionTestUtils.setField(
            service,
            "dbUrl",
            "jdbc:h2:mem:ai_sensitive_word_" + System.nanoTime() + ";MODE=MySQL;DB_CLOSE_DELAY=-1;DATABASE_TO_LOWER=TRUE");
        ReflectionTestUtils.setField(service, "dbUsername", "sa");
        ReflectionTestUtils.setField(service, "dbPassword", "");
        service.ensureTables();
    }

    @Test
    void saveAuditAndListAuditsShouldPersistBlockedEvent() {
        LocalDateTime blockedAt = ConsoleDateTimeUtil.now().withNano(0);
        AiSensitiveBlockAuditEvent event = AiSensitiveBlockAuditEvent.builder()
            .requestId("req-1")
            .routeName("ai-route-doubao.internal")
            .consumerName("consumer-a")
            .blockedAt(blockedAt)
            .blockedBy("sensitive_word")
            .requestPhase("request")
            .blockedReasonJson("{\"blocked_by\":\"sensitive_word\"}")
            .matchType("contains")
            .matchedRule("南京")
            .matchedExcerpt("请介绍南京的旅游景点")
            .build();

        AiSensitiveBlockAudit saved = service.saveAudit(event, "Demo User");

        assertNotNull(saved);
        assertNotNull(saved.getId());
        assertEquals("req-1", saved.getRequestId());
        assertEquals("ai-route-doubao.internal", saved.getRouteName());
        assertEquals("consumer-a", saved.getConsumerName());
        assertEquals("Demo User", saved.getDisplayName());
        assertEquals("request", saved.getRequestPhase());
        assertEquals("contains", saved.getMatchType());
        assertEquals("南京", saved.getMatchedRule());
        assertEquals(1, service.countAuditRecords());

        List<AiSensitiveBlockAudit> audits =
            service.listAudits("consumer-a", "Demo", "ai-route-doubao.internal", "contains", null, null, 10);

        assertEquals(1, audits.size());
        assertEquals(saved.getId(), audits.get(0).getId());
        assertEquals("请介绍南京的旅游景点", audits.get(0).getMatchedExcerpt());
        assertEquals(blockedAt, Instant.parse(audits.get(0).getBlockedAt()).atZone(ConsoleDateTimeUtil.APP_ZONE_ID)
            .toLocalDateTime());
    }

    @Test
    void listAuditsShouldSupportRfc3339TimeWindow() {
        LocalDateTime blockedAt = LocalDateTime.of(2026, 3, 31, 13, 0, 0);
        service.saveAudit(
            AiSensitiveBlockAuditEvent.builder()
                .requestId("req-rfc3339")
                .routeName("ai-route-doubao.internal")
                .consumerName("consumer-a")
                .blockedAt(blockedAt)
                .requestPhase("request")
                .matchType("contains")
                .matchedRule("南京")
                .matchedExcerpt("南京怎么样")
                .build(),
            "Demo User");

        List<AiSensitiveBlockAudit> audits = service.listAudits(
            "consumer-a",
            null,
            "ai-route-doubao.internal",
            "contains",
            blockedAt.minusMinutes(5).atZone(ConsoleDateTimeUtil.APP_ZONE_ID).toInstant().toString(),
            blockedAt.plusMinutes(5).atZone(ConsoleDateTimeUtil.APP_ZONE_ID).toInstant().toString(),
            10);

        assertEquals(1, audits.size());
        assertEquals("req-rfc3339", audits.get(0).getRequestId());
    }

    @Test
    void getAndSaveSystemConfigShouldPersistNormalizedDictionary() {
        AiSensitiveSystemConfig initial = service.getSystemConfig();
        assertNotNull(initial);
        assertEquals(Boolean.FALSE, initial.getSystemDenyEnabled());

        AiSensitiveSystemConfig saved = service.saveSystemConfig(
            AiSensitiveSystemConfig.builder()
                .systemDenyEnabled(Boolean.TRUE)
                .dictionaryText(" 天安门 \n\n天安门\n南京")
                .build(),
            "tester");

        assertNotNull(saved);
        assertEquals(Boolean.TRUE, saved.getSystemDenyEnabled());
        assertEquals("天安门\n南京", saved.getDictionaryText());
        assertEquals("tester", saved.getUpdatedBy());
        assertEquals(2, service.parseDictionaryWords(saved.getDictionaryText()).size());
    }
}
