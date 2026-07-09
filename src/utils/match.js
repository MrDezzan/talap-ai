// Helper to calculate real match percentage based on the student's AI Profile (skills, interests, top profession).
// Returns null if the user has not completed the quiz or doesn't have an AI profile.
export const calculateRealMatch = (item, portfolio, isProfession = false) => {
  if (!portfolio) return null;

  let aiResult = portfolio.ai_result;
  while (aiResult && typeof aiResult === 'string') {
    try {
      aiResult = JSON.parse(aiResult);
    } catch (e) {
      break;
    }
  }

  if (!aiResult || !aiResult.top_profession) return null;

  const topProfession = aiResult.top_profession || '';
  const interests = portfolio.interests || [];
  const skills = portfolio.skills || [];

  const itemName = (item.name || '').toLowerCase().trim();
  const lowerTopProf = topProfession.toLowerCase().trim();

  // Baseline matching score is now 15% to represent low compatibility for unrelated fields
  let score = 15;

  // Helper to map a profession name to a standard category
  const getCategoryForProfession = (profName) => {
    const ln = (profName || '').toLowerCase();
    if (ln.includes('engineer') || ln.includes('specialist') || ln.includes('analyst') || ln.includes('developer') || ln.includes('scientist')) {
      if (ln.includes('civil') || ln.includes('robotics') || ln.includes('hardware') || ln.includes('aerospace') || ln.includes('environmental')) {
        return 'Инженерия';
      }
      return 'IT и данные';
    }
    if (ln.includes('designer') || ln.includes('artist') || ln.includes('creator') || ln.includes('director') || ln.includes('animator')) {
      return 'Дизайн';
    }
    if (ln.includes('manager') || ln.includes('consultant') || ln.includes('analyst') || ln.includes('director') || ln.includes('financial')) {
      if (ln.includes('marketing') || ln.includes('product') || ln.includes('business') || ln.includes('financial') || ln.includes('consultant')) {
        return 'Бизнес';
      }
    }
    if (ln.includes('doctor') || ln.includes('biotech') || ln.includes('geneticist') || ln.includes('psychologist') || ln.includes('biochemist')) {
      return 'Медицина';
    }
    if (ln.includes('teacher') || ln.includes('consultant') || ln.includes('learning') || ln.includes('course')) {
      return 'Образование';
    }
    return 'Наука'; // default fallback
  };

  const userCategory = getCategoryForProfession(topProfession);

  if (isProfession) {
    // 1. Direct match with top recommended profession
    if (lowerTopProf && (itemName.includes(lowerTopProf) || lowerTopProf.includes(itemName))) {
      score += 75;
    } else {
      // 2. Check alternative recommended professions if present
      const alternativeProfessions = aiResult.alternative_professions || [];
      let foundAlt = false;
      alternativeProfessions.forEach(alt => {
        const altLower = alt.toLowerCase().trim();
        if (itemName.includes(altLower) || altLower.includes(itemName)) {
          score += 55;
          foundAlt = true;
        }
      });

      // 3. Category match boost (if it's in the same career path/category as top profession)
      if (!foundAlt && userCategory && item.category && userCategory.toLowerCase() === item.category.toLowerCase()) {
        score += 35;
      }
    }

    // 4. Overlap with skills & interests compared to category/tags
    let overlaps = 0;
    if (item.tags) {
      item.tags.forEach(tag => {
        const tagLower = tag.toLowerCase().trim();
        interests.forEach(interest => {
          if (interest.toLowerCase().includes(tagLower)) {
            overlaps += 1.5;
          }
        });
        skills.forEach(skill => {
          if (skill.toLowerCase().includes(tagLower)) {
            overlaps += 2;
          }
        });
      });
    }
    score += Math.round(overlaps * 3);
  } else {
    // For grants:
    // Match based on grant tags (IT, Kazakhstan, Abroad) compared to recommended profession, skills & interests
    let overlaps = 0;
    if (item.tags) {
      item.tags.forEach(tag => {
        const tagLower = tag.toLowerCase().trim();
        // If the grant matches userCategory (e.g. IT tag for IT user)
        if (userCategory === 'IT и данные' && (tagLower === 'it' || tagLower === 'ai' || tagLower === 'программирование')) {
          overlaps += 6;
        }
        if (userCategory === 'Инженерия' && (tagLower === 'инженерия' || tagLower === 'строительство' || tagLower === 'робототехника')) {
          overlaps += 6;
        }
        if (userCategory === 'Бизнес' && (tagLower === 'бизнес' || tagLower === 'маркетинг' || tagLower === 'менеджмент')) {
          overlaps += 6;
        }
        if (userCategory === 'Дизайн' && (tagLower === 'дизайн' || tagLower === 'креатив' || tagLower === 'интерфейсы')) {
          overlaps += 6;
        }

        if (lowerTopProf && (lowerTopProf.includes(tagLower) || tagLower.includes(lowerTopProf))) {
          overlaps += 4;
        }
        interests.forEach(interest => {
          if (interest.toLowerCase().includes(tagLower)) {
            overlaps += 2;
          }
        });
        skills.forEach(skill => {
          if (skill.toLowerCase().includes(tagLower)) {
            overlaps += 2;
          }
        });
      });
    }
    score += Math.round(overlaps * 4);
  }

  // Add a stable pseudo-random fluctuation based on the item name length so it doesn't look flat
  const nameHash = (itemName.length + (item.category || '').length) % 6;
  score += nameHash;

  // Bounds checking (minimum is now 10%, max is 98%)
  if (score > 98) score = 98;
  if (score < 10) score = 10;

  return score;
};
