// Reverse Engineer createLessonPlanV3 Input (Parameterized)
// Usage: cypher-safe --preset resources-dev --params '{"lpID":"lp_xxx"}' "$(cat reverse_v3.cypher)"

MATCH (lp:LessonPlan {id: $lpID})

// Get basic info
CALL {
    WITH lp
    OPTIONAL MATCH (lp)<-[:CREATED]-(u:User)
    RETURN u.id AS classId
}

CALL {
    WITH lp
    OPTIONAL MATCH (lp)-[:FOR]->(st:Subtopic)
    RETURN st.id AS subtopicID
}

// Get lessons
CALL {
    WITH lp
    MATCH (lp)-[plansRel:PLANS]->(lesson:Lesson)
    WITH lesson, plansRel
    ORDER BY plansRel.index
    
    // For each lesson, get its sections
    CALL {
        WITH lesson
        
        // Get Introduction Nodes
        CALL {
            WITH lesson
            OPTIONAL MATCH (lesson)-[:CONTAINS]->(is:IntroductionSection)-[ir:INCLUDES]->(iNode:LessonPlanNode)
            WHERE iNode.enabled = true
            WITH iNode, ir
            ORDER BY ir.index
            RETURN collect({type: iNode.type}) AS introductionNodes
        }
        
        // Get Skill Sections
        CALL {
            WITH lesson
            OPTIONAL MATCH (lesson)-[cr:CONTAINS]->(ss:SkillSection)-[:HAS]->(skill:Skill)
            WITH ss, skill, cr
            ORDER BY cr.index
            
            // For each skill section, get its nodes
            CALL {
                WITH ss
                OPTIONAL MATCH (ss)-[nr:INCLUDES]->(sNode:LessonPlanNode)
                WHERE sNode.enabled = true
                WITH sNode, nr
                ORDER BY nr.index
                
                // Build node object with conditional templateConfig
                WITH CASE 
                    WHEN sNode.type = 'SKILL_SLIDE' THEN {
                        type: sNode.type,
                        title: sNode.title,
                        templateConfig: CASE 
                            WHEN sNode.skillTemplateIDs IS NOT NULL AND sNode.generalTemplateIDs IS NOT NULL 
                            THEN {
                                skillTemplateID: sNode.skillTemplateIDs[0],
                                generalTemplateID: sNode.generalTemplateIDs[0]
                            }
                            ELSE null
                        END
                    }
                    ELSE {
                        type: sNode.type
                    }
                END AS nodeObj
                
                RETURN collect(nodeObj) AS skillNodes
            }
            
            RETURN collect({
                skillID: skill.id,
                nodes: skillNodes
            }) AS skillSections
        }
        
        // Get Practice Nodes
        CALL {
            WITH lesson
            OPTIONAL MATCH (lesson)-[:CONTAINS]->(ps:PracticeSection)-[pr:INCLUDES]->(pNode:LessonPlanNode)
            WHERE pNode.enabled = true
            WITH pNode, pr
            ORDER BY pr.index
            
            // Build practice node with conditional cardConfig
            WITH CASE
                WHEN pNode.type = 'CARD' THEN {
                    title: pNode.title,
                    cardConfig: {
                        type: 'SLIDE',
                        icon: apoc.convert.fromJsonMap(pNode.icon),
                        slideConfig: {
                            whiteboardTemplateID: pNode.whiteboardTemplateID
                        }
                    }
                }
                ELSE {
                    type: pNode.type
                }
            END AS practiceNodeObj
            
            RETURN collect(practiceNodeObj) AS practiceNodes
        }
        
        RETURN {
            duration: lesson.duration,
            introductionNodes: introductionNodes,
            skillSections: skillSections,
            practiceNodes: practiceNodes
        } AS lessonObj
    }
    
    RETURN collect(lessonObj) AS lessons
}

// Build final result
RETURN {
    classId: classId,
    input: {
        name: coalesce(lp.name, ""),
        context: coalesce(lp.context, ""),
        subtopicID: subtopicID,
        lessonType: "STANDARD",
        lessons: lessons
    }
} AS reconstructedInput
