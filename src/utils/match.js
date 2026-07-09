// Helper to calculate real match percentage based on the student's AI Profile (skills, interests, top profession).
// Returns null if the user has not completed the quiz or doesn't have an AI profile.
export const calculateRealMatch = (item, aiProfile, isProfession = false) => {
  if (!aiProfile || !aiProfile.top_profession) return null;

  const topProfession = aiProfile.top_profession || '';
  const interests = aiProfile.interests || [];
  const skills = aiProfile.skills || [];

  const itemName = (item.name || '').toLowerCase();
  const lowerTopProf = topProfession.toLowerCase();

  // Baseline matching score
  let score = 55;

  if (isProfession) {
    // 1. Direct match with top recommended profession
    if (lowerTopProf && (itemName.includes(lowerTopProf) || lowerTopProf.includes(itemName))) {
      score += 35;
    } else {
      // 2. Check alternative recommended professions if present
      const alternativeProfessions = aiProfile.alternative_professions || [];
      alternativeProfessions.forEach(alt => {
        const altLower = alt.toLowerCase();
        if (itemName.includes(altLower) || altLower.includes(itemName)) {
          score += 22;
        }
      });
    }

    // 3. Match based on skills & interests compared to category/tags
    let overlaps = 0;
    if (item.category) {
      const catLower = item.category.toLowerCase();
      interests.forEach(interest => {
        if (interest.toLowerCase().includes(catLower) || catLower.includes(interest.toLowerCase())) {
          overlaps += 2;
        }
      });
    }

    if (item.tags) {
      item.tags.forEach(tag => {
        const tagLower = tag.toLowerCase();
        interests.forEach(interest => {
          if (interest.toLowerCase().includes(tagLower) || tagLower.includes(interest.toLowerCase())) {
            overlaps += 1;
          }
        });
        skills.forEach(skill => {
          if (skill.toLowerCase().includes(tagLower) || tagLower.includes(skill.toLowerCase())) {
            overlaps += 1.5;
          }
        });
      });
    }
    score += Math.round(overlaps * 4);
  } else {
    // For grants:
    // Match based on grant tags (IT, Kazakhstan, Abroad) compared to recommended profession, skills & interests
    let overlaps = 0;
    if (item.tags) {
      item.tags.forEach(tag => {
        const tagLower = tag.toLowerCase();
        if (lowerTopProf && (lowerTopProf.includes(tagLower) || tagLower.includes(lowerTopProf))) {
          overlaps += 3;
        }
        interests.forEach(interest => {
          if (interest.toLowerCase().includes(tagLower) || tagLower.includes(interest.toLowerCase())) {
            overlaps += 1.5;
          }
        });
        skills.forEach(skill => {
          if (skill.toLowerCase().includes(tagLower) || tagLower.includes(skill.toLowerCase())) {
            overlaps += 1.5;
          }
        });
      });
    }
    score += Math.round(overlaps * 5);
  }

  // Bounds checking
  if (score > 98) score = 98;
  if (score < 45) score = 45;

  return score;
};
